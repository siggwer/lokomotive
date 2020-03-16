// Copyright 2020 The Lokomotive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO: Add AWS when prometheus operator is being installed on AWS
// +build aws packet
// +build poste2e

package monitoring

import (
	"context"
	"fmt"
	"os"
	"testing"
	"text/tabwriter"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func testScrapeTargetRechability(t *testing.T, v1api v1.API) {
	var w *tabwriter.Writer

	var err error

	// This loop ensures that we try to query prometheus for the state of targets multiple times.
	// This is the retry logic.
	for i := 0; i < 20; i++ {
		w, err = runScrapeTest(v1api)
		if err != nil && w == nil {
			t.Fatalf("%v", err)
		} else if err == nil {
			return
		}

		t.Logf("Running scrape test iteration #%d", i)
		time.Sleep(time.Minute)
	}

	t.Error("some prometheus scrape targets are down")

	// Finally print the table of all the targets that are down.
	if err := w.Flush(); err != nil {
		t.Errorf("error printing the unreachable targets: %v", err)
	}
}

// runScrapeTest queries Prometheus for the scrape targets. Once found it looks for the targets that
// are not UP. If any target is found that is not UP this test is marked as failed. This test always
// returns error if it has failed.
//
// This function also returns a list of all the targets that are not in UP state. The name of the
// target is formatted as `namespace/service-name target-status`. The list is formatted as
// tabwriter.
func runScrapeTest(v1api v1.API) (*tabwriter.Writer, error) {
	const contextTimeout = 10

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout*time.Second)
	defer cancel()

	targets, err := v1api.Targets(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing targets from prometheus: %v", err)
	}

	// Initialize the tabwriter to print the output in tabular format.
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 16, 8, 2, '\t', 0)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Service\tHealth\n")
	fmt.Fprintf(w, "-------\t------\n")

	// Boolean used to identify if tests failed.
	var testsFailed bool

	for _, target := range targets.Active {
		if target.Health == v1.HealthGood {
			continue
		}

		// This variable marks that the test has failed but we don't return from here because we
		// need the list of all the targets that are not in UP state.
		testsFailed = true

		fmt.Fprintf(w, "%s/%s\t%s\n", target.Labels["namespace"], target.Labels["service"], target.Health)
	}

	fmt.Fprintf(w, "\n")

	if testsFailed {
		return w, fmt.Errorf("some prometheus scrape targets are down")
	}

	return nil, nil
}
