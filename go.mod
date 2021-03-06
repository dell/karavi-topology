module github.com/dell/karavi-topology

go 1.14

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/mock v1.5.0
	github.com/gorilla/mux v1.8.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v0.7.0
	go.opentelemetry.io/otel/exporters/trace/zipkin v0.7.0
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f // indirect
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
)
