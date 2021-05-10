package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/klog/v2"
)

const (
	httpTimeout = 61 * time.Second
)

type Tracker struct {
	IpAddress      string `json:"ipAddress"`
	IdentityPubkey string `json:"identityPubkey"`
	GossipPort     int64  `json:"gossipPort"`
	TpuPort        int64  `json:"tpuPort"`
	Version        string `json:"version"`
	RpcHost        string `json:"rpcHost"`
}

type TrackerCollector struct {
	IpAddress      *prometheus.Desc
	IdentityPubkey *prometheus.Desc
	GossipPort     *prometheus.Desc
	TpuPort        *prometheus.Desc
	Version        *prometheus.Desc
	RpcHost        *prometheus.Desc
}

func GetGossip() ([]byte, error) {
	jsonFile, err := os.Open("/home/joe/output.json")
	if err != nil {
		return nil, fmt.Errorf("failed: %w", err)
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	return byteValue, err
}

func NewCollector() *TrackerCollector {
	return &TrackerCollector{
		IpAddress: prometheus.NewDesc(
			"gossip_ip_address",
			"IP address",
			[]string{"ip_address", "index"}, nil),
		IdentityPubkey: prometheus.NewDesc(
			"gossip_identity_pubkey",
			"Identity Pubkey",
			[]string{"pubkey", "index"}, nil),
		GossipPort: prometheus.NewDesc(
			"gossip_port",
			"Gossip port",
			[]string{"index"}, nil),
		TpuPort: prometheus.NewDesc(
			"gossip_tpu_port",
			"Tpu Port",
			[]string{"index"}, nil),
		Version: prometheus.NewDesc(
			"gossip_version",
			"Gossip Version",
			[]string{"version", "index"}, nil),
		RpcHost: prometheus.NewDesc(
			"gossip_rpc_host",
			"Rpc Host",
			[]string{"rpc_host", "index"}, nil),
	}
}

func (c *TrackerCollector) Describe(ch chan<- *prometheus.Desc) {
}

func (c *TrackerCollector) mustEmitMetrics(ch chan<- prometheus.Metric, response Tracker, index string) {
	ch <- prometheus.MustNewConstMetric(c.IpAddress, prometheus.GaugeValue, 0, response.IpAddress, index)
	ch <- prometheus.MustNewConstMetric(c.IdentityPubkey, prometheus.GaugeValue, 0, response.IdentityPubkey, index)
	ch <- prometheus.MustNewConstMetric(c.GossipPort, prometheus.GaugeValue, float64(response.GossipPort), index)
	ch <- prometheus.MustNewConstMetric(c.TpuPort, prometheus.GaugeValue, float64(response.TpuPort), index)
	ch <- prometheus.MustNewConstMetric(c.Version, prometheus.GaugeValue, 0, response.Version, index)
	ch <- prometheus.MustNewConstMetric(c.RpcHost, prometheus.GaugeValue, 0, response.RpcHost, index)
}

func (c *TrackerCollector) Collect(ch chan<- prometheus.Metric) {

	jsonData, err := GetGossip()
	if err != nil {
		klog.V(2).Infof("response: %v", err)
	}

	var gossip []Tracker

	if err = json.Unmarshal(jsonData, &gossip); err != nil {
		klog.V(2).Infof("failed to decode response body: %w", err)
	}

	for t_index, tracker := range gossip {
		_, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		index := strconv.Itoa(t_index)

		if err != nil {
			ch <- prometheus.MustNewConstMetric(c.IpAddress, prometheus.GaugeValue, 0, err.Error(), index)
			ch <- prometheus.MustNewConstMetric(c.IdentityPubkey, prometheus.GaugeValue, 0, err.Error(), index)
			ch <- prometheus.MustNewConstMetric(c.GossipPort, prometheus.GaugeValue, float64(-1), index)
			ch <- prometheus.MustNewConstMetric(c.TpuPort, prometheus.GaugeValue, float64(-1), index)
			ch <- prometheus.MustNewConstMetric(c.Version, prometheus.GaugeValue, 0, err.Error(), index)
			ch <- prometheus.MustNewConstMetric(c.RpcHost, prometheus.GaugeValue, 0, err.Error(), index)
		} else {
			c.mustEmitMetrics(ch, tracker, index)
		}
	}
}

func main() {

	Collector := NewCollector()

	prometheus.MustRegister(Collector)

	http.Handle("/metrics", promhttp.Handler())

	klog.Infof("listening on %s", "8080")
	klog.Fatal(http.ListenAndServe(":8080", nil))
}

