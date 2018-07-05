{
  global: {
  },
  components: {
    ss: {
      containerPort: 8090,
      image: 'ss',
      name: 'ss',
      replicas: 1,
      servicePort: 80,
      nodePort: 31001,
      type: 'NodePort',
    },
    'bomb-squad': {
      containerPort: 8080,
      image: 'bomb-squad',
      imageTag: 'latest',
      name: 'bomb-squad',
      replicas: 1,
      servicePort: 80,
      nodePort: 31002,
      type: 'NodePort',
    },
    prometheus: {
      name: 'prometheus',
      replicas: 1,
      promImage: 'prom/prometheus:v2.2.1',
      prometheusServicePort: 9090,
      prometheusTargetPort: 9090,
      prometheusNodePort: 30090,
      promRules: |||
        groups:
        - name: card_counter
          interval: 10s
          rules:
          - record: card_count
            expr: label_replace( count by(__name__) ({__name__!="", __name__!="card_count"}), "metric_name", "$1", "__name__", "(.+)" )
          - record: card_count:rate5m
            expr: rate(card_count[5m])
      |||,
      promConfig: |||
        remote_write:
        rule_files:
        - /etc/config/rules.yml
        global:
          scrape_interval: 10s
          scrape_timeout: 10s
          evaluation_interval: 1m
        scrape_configs:
        - job_name: prometheus
          scrape_interval: 10s
          scrape_timeout: 10s
          metrics_path: /metrics
          scheme: http
          static_configs:
          - targets:
            - localhost:9090
        - job_name: bomb-squad
          scrape_interval: 10s
          scrape_timeout: 10s
          metrics_path: /metrics
          scheme: http
          static_configs:
          - targets:
            - localhost:8080
        - job_name: kubernetes-apiservers
          scrape_interval: 10s
          scrape_timeout: 10s
          metrics_path: /metrics
          scheme: https
          kubernetes_sd_configs:
          - api_server: null
            role: endpoints
            namespaces:
              names: []
          bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
          tls_config:
            ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
            insecure_skip_verify: true
          relabel_configs:
          - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
            separator: ;
            regex: default;kubernetes;https
            replacement: $1
            action: keep

        - job_name: 'ft-kubernetes-pods'
          kubernetes_sd_configs:
            - role: pod
          relabel_configs:
            - source_labels: [__meta_kubernetes_pod_annotation_freshtracks_io_scrape]
              action: keep
              regex: true
            - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
              action: replace
              target_label: __metrics_path__
              regex: (.+)
            - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
              action: replace
              regex: (.+):(?:\d+);(\d+)
              replacement: ${1}:${2}
              target_label: __address__
            - action: drop
              source_labels: [__meta_kubernetes_pod_container_port_name]
              regex: .*-noscrape
            - action: labelmap
              regex: __meta_kubernetes_pod_label_(.+)
            - source_labels: [__meta_kubernetes_namespace]
              action: replace
              target_label: namespace
            - source_labels: [__meta_kubernetes_namespace]
              action: replace
              target_label: ft_namespace
            - source_labels: [__meta_kubernetes_pod_name]
              action: replace
              target_label: pod_name
            - action: replace
              source_labels: [pod_name]
              target_label: replica_set
              regex: ^(.+?)((-[a-z0-9]+)-[a-z0-9]{5})$
              replacement: '${1}${3}'
            - action: replace
              source_labels: [pod_name]
              target_label: ft_workload
              regex: ^(.+?)((-[a-z0-9]+)?-[a-z0-9]{5})$
              replacement: '${1}'
            - action: replace
              source_labels: [pod_name]
              target_label: ft_pod
            - action: replace
              source_labels: [pod_name, __meta_kubernetes_pod_container_name]
              target_label: ft_container
            - source_labels: [__meta_kubernetes_pod_node_name]
              separator: ;
              regex: (.*)
              target_label: node
              replacement: $1
              action: replace
            - action: replace
              source_labels: [node]
              target_label: ft_node

        - job_name: 'ft-kube-state-metrics'
          honor_labels: true
          scheme: http
          tls_config:
            ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
            insecure_skip_verify: true
          bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
          kubernetes_sd_configs:
            - role: pod
          relabel_configs:
            - source_labels: [__meta_kubernetes_pod_label_component]
              action: keep
              regex: freshtracks-kube-state-metrics
            - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
              action: replace
              target_label: __metrics_path__
              regex: (.+)
            - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
              action: replace
              regex: (.+):(?:\d+);(\d+)
              replacement: ${1}:${2}
              target_label: __address__
            - action: labelmap
              regex: __meta_kubernetes_pod_label_(.+)
          metric_relabel_configs:
            - source_labels: [namespace]
              action: replace
              target_label: ft_namespace
            - source_labels: [pod]
              action: replace
              target_label: ft_pod
            - source_labels: [ft_pod]
              target_label: ft_replicaset
              regex: ^(.+?)((-[a-z0-9]+)-[a-z0-9]{5})$
              replacement: '${1}${3}'
              action: replace
            - source_labels: [ft_replicaset, replicaset]
              target_label: ft_replicaset
              regex: ^;(.+)$
              replacement: '${1}'
              action: replace
            - source_labels: [ft_pod]
              action: replace
              target_label: ft_workload
              regex: ^(.+?)((-[a-z0-9]+)?-[a-z0-9]{5})$
              replacement: '${1}'
            - source_labels: [ft_workload, replicaset]
              target_label: ft_workload
              regex: ^;(.+)-[^-]+$
              replacement: '${1}'
              action: replace
            - source_labels: [pod_name, container]
              action: replace
              target_label: ft_container
            - source_labels: [node]
              target_label: ft_node
              action: replace

        - job_name: 'ft-kubernetes-cadvisor'
          scheme: https
          tls_config:
            ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
            insecure_skip_verify: true
          bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
          kubernetes_sd_configs:
            - role: node
          relabel_configs:
            - action: labelmap
              regex: __meta_kubernetes_node_label_(.+)
            - action: replace
              source_labels: [__address__]
              target_label: instance
            - target_label: __address__
              replacement: kubernetes.default.svc:443
            - source_labels: [__meta_kubernetes_node_name]
              regex: (.+)
              target_label: __metrics_path__
              replacement: /api/v1/nodes/${1}:4194/proxy/metrics
          metric_relabel_configs:
            - action: replace
              source_labels: [id]
              regex: '^/machine\.slice/machine-rkt\\x2d([^\\]+)\\.+/([^/]+)\.service$'
              target_label: rkt_container_name
              replacement: '${2}-${1}'
            - action: replace
              source_labels: [pod_name]
              target_label: replica_set
              regex: ^(.+?)((-[a-z0-9]+)-[a-z0-9]{5})$
              replacement: '${1}${3}'
            - action: replace
              source_labels: [pod_name]
              target_label: ft_workload
              regex: ^(.+?)((-[a-z0-9]+)?-[a-z0-9]{5})$
              replacement: '${1}'
            - action: replace
              source_labels: [pod_name]
              target_label: ft_pod
            - action: replace
              source_labels: [pod_name, container_name]
              target_label: ft_container
            - action: replace
              source_labels: [namespace]
              target_label: ft_namespace
            - action: drop
              source_labels: [container_name]
              regex: ^$
            - action: replace
              source_labels: [kubernetes_io_hostname]
              target_label: ft_node
      |||,
    },
  },
}
