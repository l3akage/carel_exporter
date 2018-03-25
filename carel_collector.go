package main

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/soniah/gosnmp"
)

const prefix = "carel_"

var (
	upDesc             *prometheus.Desc
	waterOutletDesc    *prometheus.Desc
	waterInletDesc     *prometheus.Desc
	airTempDesc        *prometheus.Desc
	fanSpeedDesc       *prometheus.Desc
	compressorFreqDesc *prometheus.Desc
)

func init() {
	l := []string{"target"}
	upDesc = prometheus.NewDesc(prefix+"up", "Scrape of target was successful", l, nil)
	waterOutletDesc = prometheus.NewDesc(prefix+"water_outlet_temp", "Water outlet temperature", l, nil)
	waterInletDesc = prometheus.NewDesc(prefix+"water_inlet_temp", "Water inlet temperature", l, nil)
	airTempDesc = prometheus.NewDesc(prefix+"air_temp", "Air temperature", l, nil)
	fanSpeedDesc = prometheus.NewDesc(prefix+"fan_speed", "Fan speed", l, nil)
	compressorFreqDesc = prometheus.NewDesc(prefix+"compressor_frequency", "Compressor frequency", l, nil)
}

type carelCollector struct {
}

func (c carelCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- waterOutletDesc
	ch <- waterInletDesc
	ch <- airTempDesc
	ch <- fanSpeedDesc
	ch <- compressorFreqDesc
}

func (c carelCollector) collectTarget(target string, ch chan<- prometheus.Metric, wg *sync.WaitGroup) {
	defer wg.Done()
	snmp := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161,
		Community: *snmpCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(2) * time.Second,
	}
	err := snmp.Connect()
	if err != nil {
		log.Infof("Connect() err: %v\n", err)
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0, target)
		return
	}
	defer snmp.Conn.Close()

	oids := []string{"1.3.6.1.4.1.9839.2.1.2.3.0", "1.3.6.1.4.1.9839.2.1.2.4.0", "1.3.6.1.4.1.9839.2.1.2.5.0",
		"1.3.6.1.4.1.9839.2.1.2.15.0", "1.3.6.1.4.1.9839.2.1.2.164.0"}
	result, err2 := snmp.Get(oids)
	if err2 != nil {
		log.Infof("Get() err: %v\n", err)
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0, target)
		return
	}
	for _, variable := range result.Variables {
		if variable.Value == nil {
			continue
		}
		switch variable.Name[1:] {
		case oids[0]:
			ch <- prometheus.MustNewConstMetric(waterOutletDesc, prometheus.GaugeValue, float64(variable.Value.(int))/10, target)
		case oids[1]:
			ch <- prometheus.MustNewConstMetric(waterInletDesc, prometheus.GaugeValue, float64(variable.Value.(int))/10.0, target)
		case oids[2]:
			ch <- prometheus.MustNewConstMetric(airTempDesc, prometheus.GaugeValue, float64(variable.Value.(int))/10.0, target)
		case oids[3]:
			ch <- prometheus.MustNewConstMetric(fanSpeedDesc, prometheus.GaugeValue, float64(variable.Value.(int))/10.0, target)
		case oids[4]:
			ch <- prometheus.MustNewConstMetric(compressorFreqDesc, prometheus.GaugeValue, float64(variable.Value.(int))/10.0, target)
		}
	}
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1, target)
}

func (c carelCollector) Collect(ch chan<- prometheus.Metric) {
	targets := strings.Split(*snmpTargets, ",")
	wg := &sync.WaitGroup{}

	for _, target := range targets {
		wg.Add(1)
		go c.collectTarget(target, ch, wg)
	}

	wg.Wait()
}
