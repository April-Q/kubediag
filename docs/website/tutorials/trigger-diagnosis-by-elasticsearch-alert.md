# 通过 ElasticsearchQuery 消息触发诊断

本文介绍了如何通过 ElasticsearchQuery 查询 Elasticsearch 日志创建 Diagnosis 来触发诊断。

## 开始之前

在教程开始前，您需要确定 Kubernetes 集群中已经正确安装 KubeDiag。

## 在 KubeDiag Master 参数中指定需要的 Elasticsearch

您需要在 KubeDiag Master 启动时指定下列参数以使用该功能：

| 参数 | 类型 | 描述 | 示例 |
|-|-|-|-|
| --es-url | strings | 需要连接 Elasticsearch 集群的地址列表。 | 127.0.0.1:9092,127.0.0.2:9092 |
| --es-username | string | 登录 Elasticsearch 的用户名。 | elastic |
| --es-password | string | 登录 Elasticsearch 的密码。 | elasticpassword |

如果上述参数均未指定，则通过 ElasticsearchQuery 查询日志触发诊断的功能不开启。

## 举例说明

创建一个 Trigger，在 `ruleconfig` 中描述查询语句：

```yaml
apiVersion: diagnosis.kubediag.org/v1
kind: Trigger
metadata:
  name: elatiscsearch-example
spec:
  operationSet: elatiscsearch-example
  sourceTemplate:
    elasticSearchAlertTemplate:
      nodeNameReferenceLabel: kubernetes.node.name
      podNameReferenceLabel: kubernetes.pod.name
      podNamespaceReferenceLabel: kubernetes.namespace
      containerReferenceLabel: kubernetes.container.name
      parameterInjectionLabels:
      - log.file.path
      ruleconfig:
        index: "test"
        schedule: "@every 2m"
        body: '{
  "query": {
    "bool": {
      "must": {
        "match": {
          "stream": "stderr"
        }
      },
      "filter": {
        "range": {
          "@timestamp": {
            "gt": "now-2m"
          }
        }
      }
    }
  },
  "from": 1,
  "size": 1
}'
```

根据以上 `body` 查询出来的结果如下：

```json

{
  // ......

  "hits" : {
    "total" : {
      "value" : 23,
      "relation" : "eq"
    },
    "max_score" : 0.053220153,
    "hits" : [
      {
        "_index" : "filebeat-8.0.0-2021.07.09-000001",
        "_id" : "ILAzinoBeIRj7XR6G4FS",
        "_score" : 0.053220153,
        "_source" : {
          "@timestamp" : "2021-07-09T07:37:21.581Z",
          "ecs" : {
            "version" : "1.10.0"
          },
          "log" : {
            "offset" : 367696,
            "file" : {
              "path" : "/var/log/containers/kube-apiserver-my-node_kube-system_kube-apiserver-2bfcb139999c2bd7c7a53d08bbe12ba814775758e3085c9f77464e218955ea78.log"
            }
          },
          "message" : "I0709 07:37:21.5812021 client.go:360] parsed scheme: \"passthrough\"",
          "input" : {
            "type" : "container"
          },
          "container" : {
            "runtime" : "docker",
            "image" : {
              "name" : "k8s.gcr.io/kube-apiserver:v1.19.12"
            },
            "id" : "2bfcb139999c2bd7c7a53d08bbe12ba814775758e3085c9f77464e218955ea78"
          },
          "kubernetes" : {
            "labels" : {
              "component" : "kube-apiserver",
              "tier" : "control-plane"
            },
            "container" : {
              "name" : "kube-apiserver"
            },
            "node" : {
              "name" : "my-node",
              "uid" : "ebdaadf7-b055-4859-a3fc-97c433b6169c",
              "labels" : {
                "node-role_kubernetes_io/master" : "",
                "beta_kubernetes_io/arch" : "amd64",
                "beta_kubernetes_io/os" : "linux",
                "kubernetes_io/arch" : "amd64",
                "kubernetes_io/hostname" : "my-node",
                "kubernetes_io/os" : "linux"
              },
              "hostname" : "my-node"
            },
            "namespace_uid" : "b4200416-cca9-4370-b01a-24bb74a90d25",
            "pod" : {
              "ip" : "10.0.2.15",
              "name" : "kube-apiserver-my-node",
              "uid" : "5b3cce30-6837-4748-b2bb-c719ea71c5db"
            },
            "namespace" : "kube-system"
          },
          "host" : {
            "name" : "helm-filebeat-security-filebeat-5fv4l"
          },
          "agent" : {
            "type" : "filebeat",
            "version" : "8.0.0",
            "ephemeral_id" : "9f68933d-41b7-4fd5-8be9-cb6a09e09e4b",
            "id" : "04cf4ce1-6c1b-4256-8d3a-2952e51f0c55",
            "name" : "helm-filebeat-security-filebeat-5fv4l"
          },
          "stream" : "stderr"
        }
      }
    ]
  }
}

```

KubeDiag 匹配到消息时会创建下列 Diagnosis：

```yaml
apiVersion: diagnosis.kubediag.org/v1
kind: Diagnosis
metadata:
  labels:
    adjacency-list-hash: 57db4d79b7
  name: elasticsearch-query.elatiscsearch-example.94df165
  namespace: kubediag
spec:
  nodeName: my-node
  operationSet: elatiscsearch-example
  parameters:
    hits.log.file.path: "/var/log/containers/kube-apiserver-my-node_kube-system_kube-apiserver-2bfcb139999c2bd7c7a53d08bbe12ba814775758e3085c9f77464e218955ea78.log"
  podReference:
    container: kube-apiserver
    name: kube-apiserver-my-node
    namespace: kube-system
```
