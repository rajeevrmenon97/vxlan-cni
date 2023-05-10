FROM golang:1.20.4
COPY plugin/main.go /usr/src/vxlan-cni/plugin/
COPY go.mod /usr/src/vxlan-cni
COPY go.sum /usr/src/vxlan-cni
WORKDIR /usr/src/vxlan-cni
RUN go build -o /opt/cni/bin/vxlan plugin/main.go