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

// +build aws packet
// +build poste2e

package monitoring

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	testutil "github.com/kinvolk/lokomotive/test/components/util"
)

type AlertTestCase struct {
	Name      string
	platforms []testutil.Platform
	Alerts    []string
}

func testComponentAlerts(t *testing.T, v1api v1.API) {
	alertTestCases := []AlertTestCase{
		{
			Name:      "metallb-rules",
			platforms: []testutil.Platform{testutil.PlatformPacket},
			Alerts: []string{
				"MetalLBNoBGPSession", "MetalLBConfigStale", "MetalLBControllerPodsAvailability",
				"MetalLBSpeakerPodsAvailability",
			},
		},
	}

	var err error

	const (
		waitTime = 30
		retries  = 20
	)

	for i := 0; i < retries; i++ {
		t.Logf("iteration: #%d", i)

		if err = runAlertTest(t, v1api, alertTestCases); err == nil {
			return
		}

		t.Logf("failed with error: %v.\t retrying...", err)
		time.Sleep(waitTime * time.Second)
	}
	t.Errorf("failed with error: %v", err)
}

func runAlertTest(t *testing.T, v1api v1.API, testCases []AlertTestCase) error {
	const contextTimeout = 10

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout*time.Second)
	defer cancel()

	result, err := v1api.Rules(ctx)
	if err != nil {
		return fmt.Errorf("error listing rules: %v", err)
	}

	// This map will store information from cluster so that it is easier to search it against
	// the test cases
	ruleGroups := make(map[string][]string, len(result.Groups))

	for _, ruleGroup := range result.Groups {
		rules := make([]string, 0)

		for _, rule := range ruleGroup.Rules {
			switch v := rule.(type) {
			case v1.AlertingRule:
				rules = append(rules, v.Name)
			default:
			}
		}

		ruleGroups[ruleGroup.Name] = rules
	}

	for _, alertTestCase := range testCases {
		if !testutil.IsPlatformSupported(t, alertTestCase.platforms) {
			continue
		}

		t.Logf("checking existence of %s", alertTestCase.Name)

		rules, ok := ruleGroups[alertTestCase.Name]
		if !ok {
			return fmt.Errorf("RuleGroup %q not found", alertTestCase.Name)
		}

		if !reflect.DeepEqual(rules, alertTestCase.Alerts) {
			return fmt.Errorf("Rules don't match. Expected: %#v and \ngot %#v", alertTestCase.Alerts, rules)
		}
	}

	return nil
}
