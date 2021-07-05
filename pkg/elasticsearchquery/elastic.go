package elasticsearchquery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	diagnosisv1 "github.com/kubediag/kubediag/api/v1"
)

// elasticsearchQuery manages elasticsearch query received by kubediag.
type elasticsearchQuery struct {
	// esclient prefrom operations on elasticsearch.
	esclient elasticsearch.Client
	// Context carries values across API boundaries.
	context.Context
	// Logger represents the ability to log messages.
	logr.Logger
	// client knows how to perform CRUD operations on Kubernetes objects.
	client client.Client
	// elasticsearchQueryChan recieve elasticsearch query trigger.
	elasticsearchQueryChan chan types.NamespacedName
	// queryHandlerMap hold the queryHandlers is running.
	queryHandlerMap map[string]*queryHandler
	// lock guards writes to queryHandlerMap.
	lock sync.Mutex
	// elasticsearchQueryEnabled indicates whether elasticsearchquery is enabled.
	elasticsearchQueryEnabled bool
}

// NewElasticsearchQuery create a new elasticsearchQuery.
func NewElasticsearchQuery(url []string, username, password string,
	log logr.Logger, ctx context.Context, cli client.Client,
	elasticsearchQueryChan chan types.NamespacedName, elasticsearchQueryEnabled bool) (*elasticsearchQuery, error) {

	cfg := elasticsearch.Config{
		Username:  username,
		Password:  password,
		Addresses: url,
	}
	// set the InsecureSkipVerify to true in case of self-signed certificates.
	transport := http.DefaultTransport
	tlsClientConfig := &tls.Config{InsecureSkipVerify: true}
	transport.(*http.Transport).TLSClientConfig = tlsClientConfig
	cfg.Transport = transport

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	res, err := es.Info()
	if err != nil {
		return nil, err
	}

	// Check response status
	if res.IsError() {
		return nil, fmt.Errorf("Error in check elasticsearch client status: %s", res.String())
	}

	return &elasticsearchQuery{
		esclient:                  *es,
		Context:                   ctx,
		Logger:                    log,
		client:                    cli,
		queryHandlerMap:           make(map[string]*queryHandler),
		elasticsearchQueryChan:    elasticsearchQueryChan,
		elasticsearchQueryEnabled: elasticsearchQueryEnabled,
	}, nil
}

// Run runs the elasticsearchQuery.
func (es *elasticsearchQuery) Run(stopCh chan struct{}) {
	if !es.elasticsearchQueryEnabled {
		es.Info("elasticsearchQuery is not enabled")
		return
	}

	for {
		select {
		case <-stopCh:
			for _, qh := range es.queryHandlerMap {
				es.stopQueryHandler(qh)
			}
			es.Info("stopped all elasticsearch queryhandlers.")
			return
		case alert := <-es.elasticsearchQueryChan:
			trigger, err := es.getTrigger(alert.Name)
			if err != nil && !apierrors.IsNotFound(err) {
				es.Error(err, "cannot get the trigger", "trigger", trigger.Name)
				break
			}

			// trigger is updated or deleted, stop the queryhandler if exist.
			qhInStore, ok := es.queryHandlerMap[trigger.Name]
			if ok {
				es.Info("stop queryhandler", "trigger", trigger.Name)
				es.stopQueryHandler(qhInStore)
			}
			// trigger is deleted, break now.
			if apierrors.IsNotFound(err) {
				break
			}

			// only process when trigger type is elasticsearch alert.
			if trigger.Spec.SourceTemplate.ElasticSearchQueryTemplate != nil {
				// trigger is updated or created.
				es.Info("start queryhandler", "trigger", trigger.Name)
				es.startQueryHandlerWithTrigger(trigger)
			}

		}
	}

}

// startQueryHandlerWithTrigger start a queryhandler with a trigger.
func (es *elasticsearchQuery) startQueryHandlerWithTrigger(trigger *diagnosisv1.Trigger) {
	qh := &queryHandler{
		stopCh:   make(chan struct{}),
		Logger:   es.Logger.WithName("queryhandler").WithValues("trigger", trigger.Name),
		client:   es.client,
		esclient: es.esclient,
		Context:  context.Background(),
		trigger:  trigger,
	}

	es.lock.Lock()
	es.queryHandlerMap[trigger.Name] = qh
	es.lock.Unlock()

	go func() {
		err := qh.run()
		if err != nil {
			es.Error(err, "Error in run queryhandler.", "trigger", trigger.Name)
		}
		es.lock.Lock()
		delete(es.queryHandlerMap, trigger.Name)
		es.lock.Unlock()
	}()

}

// getTrigger get a trigger from cache.
func (es *elasticsearchQuery) getTrigger(name string) (*diagnosisv1.Trigger, error) {
	var trigger diagnosisv1.Trigger
	if err := es.client.Get(es, types.NamespacedName{Name: name}, &trigger); err != nil {
		return nil, err
	}

	return &trigger, nil
}

// stopQueryHandler stop a queryhandler.
func (es *elasticsearchQuery) stopQueryHandler(qh *queryHandler) {
	qh.stopCh <- struct{}{}
}
