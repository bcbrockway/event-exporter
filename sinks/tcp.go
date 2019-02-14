/*
Copyright 2018 Alauda Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sinks

import (
	"github.com/golang/glog"
	api_v1 "k8s.io/api/core/v1"
	"net"
	"time"
	//"encoding/json"
	"fmt"
)

type TCPConf struct {
	SinkCommonConf
	Endpoint *string
}

type TCPSink struct {
	config          *TCPConf
	beforeFirstList bool
	currentBuffer   []*api_v1.Event
	logEntryChannel chan *api_v1.Event
	// Channel for controlling how many requests are being sent at the same
	// time. It's empty initially, each request adds an object at the start
	// and takes it out upon completion. Channel's capacity is set to the
	// maximum level of parallelism, so any extra request will lock on addition.
	concurrencyChannel chan struct{}
	timer              *time.Timer
	fakeTimeChannel    chan time.Time
}

func DefaultTCPConf() *TCPConf {
	return &TCPConf{
		SinkCommonConf: SinkCommonConf{
			FlushDelay:     defaultFlushDelay,
			MaxBufferSize:  defaultMaxBufferSize,
			MaxConcurrency: defaultMaxConcurrency,
		},
	}
}

func NewTCPSink(config *TCPConf) (*TCPSink, error) {

	return &TCPSink{
		beforeFirstList:    true,
		logEntryChannel:    make(chan *api_v1.Event, config.MaxBufferSize),
		config:             config,
		currentBuffer:      []*api_v1.Event{},
		timer:              nil,
		fakeTimeChannel:    make(chan time.Time),
		concurrencyChannel: make(chan struct{}, config.MaxConcurrency),
	}, nil
}

func (h *TCPSink) OnAdd(event *api_v1.Event) {
	ReceivedEntryCount.WithLabelValues(event.Source.Component).Inc()
	glog.Infof("OnAdd %v", event)
	h.logEntryChannel <- event
}

func (h *TCPSink) OnUpdate(oldEvent *api_v1.Event, newEvent *api_v1.Event) {
	var oldCount int32
	if oldEvent != nil {
		oldCount = oldEvent.Count
	}

	if newEvent.Count != oldCount+1 {
		// Sink doesn't send a LogEntry to Stackdriver, b/c event compression might
		// indicate that part of the watch history was lost, which may result in
		// multiple events being compressed. This may create an unecessary
		// flood in Stackdriver. Also this is a perfectly valid behavior for the
		// configuration with empty backing storage.
		glog.V(2).Infof("Event count has increased by %d != 1.\n"+
			"\tOld event: %+v\n\tNew event: %+v", newEvent.Count-oldCount, oldEvent, newEvent)
	}
	glog.Infof("OnUpdate %v", newEvent)

	ReceivedEntryCount.WithLabelValues(newEvent.Source.Component).Inc()

	h.logEntryChannel <- newEvent
}

func (h *TCPSink) OnDelete(*api_v1.Event) {
	// Nothing to do here
}

func (h *TCPSink) OnList(list *api_v1.EventList) {
	// Nothing to do else
	glog.Infof("OnList %v", list)
	if h.beforeFirstList {
		h.beforeFirstList = false
	}
}

func (h *TCPSink) Run(stopCh <-chan struct{}) {
	glog.Info("Starting TCP sink")
	for {
		select {
		case entry := <-h.logEntryChannel:
			h.currentBuffer = append(h.currentBuffer, entry)
			if len(h.currentBuffer) >= h.config.MaxBufferSize {
				h.flushBuffer()
			} else if len(h.currentBuffer) == 1 {
				h.setTimer()
			}
			break
		case <-h.getTimerChannel():
			h.flushBuffer()
			break
		case <-stopCh:
			glog.Info("TCP sink recieved stop signal, waiting for all requests to finish")
			glog.Info("All requests to TCP finished, exiting TCP sink")
			return
		}
	}
}

func (h *TCPSink) flushBuffer() {
	entries := h.currentBuffer
	h.currentBuffer = nil
	h.concurrencyChannel <- struct{}{}
	go h.sendEntries(entries)
}

func (h *TCPSink) sendEntries(entries []*api_v1.Event) {
	glog.V(4).Infof("Sending %d entries to TCP endpoint", len(entries))

	if err := doTCPRequest(h.config, entries); err != nil {
		// TODO how to recovery?
		fmt.Errorf("Not a expected: %v.", err)
		FailedSentEntryCount.Add(float64(len(entries)))
	} else {
		SuccessfullySentEntryCount.Add(float64(len(entries))) //
	}

	<-h.concurrencyChannel

	glog.V(4).Infof("Successfully sent %d entries to TCP endpoint", len(entries))
}

func (h *TCPSink) setTimer() {
	if h.timer == nil {
		h.timer = time.NewTimer(h.config.FlushDelay)
	} else {
		h.timer.Reset(h.config.FlushDelay)
	}
}

func (h *TCPSink) getTimerChannel() <-chan time.Time {
	if h.timer == nil {
		return h.fakeTimeChannel
	}
	return h.timer.C
}

func doTCPRequest(config *TCPConf, entries []*api_v1.Event) error {
	// Create req
	conn, err := net.Dial("tcp", *config.Endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, data := range entries {
		// Convert to json string
		params, err := encodeData(data)
		if err != nil {
			return err
		}
		conn.Write(params.Bytes())
		conn.Write([]byte("\n"))
	}

	return nil
}
