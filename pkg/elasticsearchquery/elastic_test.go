package elasticsearchquery

import (
	"fmt"
	"testing"

	diagnosisv1 "github.com/kubediag/kubediag/api/v1"
)

func TestParseBody(t *testing.T) {
	tests := []struct {
		ruleconfig diagnosisv1.ElasticSearchRuleConfig
	}{
		{
			ruleconfig: diagnosisv1.ElasticSearchRuleConfig{
				CronSchedule: "@every 2m",
				Index:        "test",
				Body: `{
						"query": {
						  "match": {
							"title": "alert"
						  }
						  }
					  }`,
			},
		},
	}
	// url := []string{"https://observability-deployment-d6bc06.es.eastus2.azure.elastic-cloud.com:9243"}
	// username := "elastic"
	// password := "jH9b50Rk2MvZsns4sgt5TYez"
	// setupLog := ctrl.Log.WithName("estest")
	// esclient, err := NewElasticSearchAlert(url, username, password, setupLog,
	// 	context.Background(), nil, nil, true)

	// if err != nil {
	// 	assert.NoError(t, err)
	// }
	for _, test := range tests {
		fmt.Print(test.ruleconfig)

	}
}
