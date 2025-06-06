package capability

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/sets"

	configv1 "github.com/openshift/api/config/v1"
)

func TestSetCapabilities(t *testing.T) {
	tests := []struct {
		name            string
		config          *configv1.ClusterVersion
		wantKnownKeys   []configv1.ClusterVersionCapability
		wantEnabledKeys []configv1.ClusterVersionCapability
	}{
		{name: "capabilities nil",
			config: &configv1.ClusterVersion{},
			// wantKnownKeys and wantEnabledKeys will be set to default set of capabilities by test
		},
		{name: "capabilities set not set",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{},
				},
			},
			// wantKnownKeys and wantEnabledKeys will be set to default set of capabilities by test
		},
		{name: "set capabilities None",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{
						BaselineCapabilitySet: configv1.ClusterVersionCapabilitySetNone,
					},
				},
			},
			wantKnownKeys:   configv1.KnownClusterVersionCapabilities,
			wantEnabledKeys: []configv1.ClusterVersionCapability{},
		},
		{name: "set capabilities 4_11",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{
						BaselineCapabilitySet:         configv1.ClusterVersionCapabilitySet4_11,
						AdditionalEnabledCapabilities: []configv1.ClusterVersionCapability{},
					},
				},
			},
			wantKnownKeys: configv1.KnownClusterVersionCapabilities,
			wantEnabledKeys: []configv1.ClusterVersionCapability{
				configv1.ClusterVersionCapabilityBaremetal,
				configv1.ClusterVersionCapabilityMarketplace,
				configv1.ClusterVersionCapabilityOpenShiftSamples,
				configv1.ClusterVersionCapabilityMachineAPI,
			},
		},
		{name: "set capabilities vCurrent",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{
						BaselineCapabilitySet:         configv1.ClusterVersionCapabilitySetCurrent,
						AdditionalEnabledCapabilities: []configv1.ClusterVersionCapability{},
					},
				},
			},
			wantKnownKeys:   configv1.KnownClusterVersionCapabilities,
			wantEnabledKeys: configv1.ClusterVersionCapabilitySets[configv1.ClusterVersionCapabilitySetCurrent],
		},
		{name: "set capabilities None with additional",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{
						BaselineCapabilitySet:         configv1.ClusterVersionCapabilitySetNone,
						AdditionalEnabledCapabilities: []configv1.ClusterVersionCapability{"cap1", "cap2", "cap3"},
					},
				},
			},
			wantKnownKeys:   configv1.KnownClusterVersionCapabilities,
			wantEnabledKeys: []configv1.ClusterVersionCapability{"cap1", "cap2", "cap3"},
		},
		{name: "set capabilities 4_11 with additional",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{
						BaselineCapabilitySet:         configv1.ClusterVersionCapabilitySet4_11,
						AdditionalEnabledCapabilities: []configv1.ClusterVersionCapability{"cap1", "cap2", "cap3"},
					},
				},
			},
			wantKnownKeys: configv1.KnownClusterVersionCapabilities,
			wantEnabledKeys: []configv1.ClusterVersionCapability{
				configv1.ClusterVersionCapabilityBaremetal,
				configv1.ClusterVersionCapabilityMarketplace,
				configv1.ClusterVersionCapabilityOpenShiftSamples,
				configv1.ClusterVersionCapabilityMachineAPI,
				"cap1",
				"cap2",
				"cap3",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caps := SetCapabilities(test.config, nil)
			if test.config.Spec.Capabilities == nil || (test.config.Spec.Capabilities != nil &&
				len(test.config.Spec.Capabilities.BaselineCapabilitySet) == 0) {

				test.wantKnownKeys = configv1.KnownClusterVersionCapabilities
				test.wantEnabledKeys = configv1.ClusterVersionCapabilitySets[configv1.ClusterVersionCapabilitySetCurrent]
			}
			if len(caps.Known) != len(test.wantKnownKeys) {
				t.Errorf("Incorrect number of Known keys, wanted: %q. Known returned: %v", test.wantKnownKeys, caps.Known)
			}
			for _, v := range test.wantKnownKeys {
				if _, ok := caps.Known[v]; !ok {
					t.Errorf("Missing Known key %q. Known returned : %v", v, caps.Known)
				}
			}
			if len(caps.Enabled) != len(test.wantEnabledKeys) {
				t.Errorf("Incorrect number of Enabled keys, wanted: %q. Enabled returned: %v", test.wantEnabledKeys, caps.Enabled)
			}
			for _, v := range test.wantEnabledKeys {
				if _, ok := caps.Enabled[v]; !ok {
					t.Errorf("Missing Enabled key %q. Enabled returned : %v", v, caps.Enabled)
				}
			}
		})
	}
}

func TestSetCapabilitiesWithImplicitlyEnabled(t *testing.T) {
	tests := []struct {
		name         string
		config       *configv1.ClusterVersion
		wantImplicit sets.Set[configv1.ClusterVersionCapability]
		priorEnabled []configv1.ClusterVersionCapability
	}{
		{name: "set baseline capabilities with additional",
			config: &configv1.ClusterVersion{
				Spec: configv1.ClusterVersionSpec{
					Capabilities: &configv1.ClusterVersionCapabilitiesSpec{
						BaselineCapabilitySet: configv1.ClusterVersionCapabilitySetCurrent,
					},
				},
			},
			priorEnabled: []configv1.ClusterVersionCapability{"cap1", "cap2", "cap3", "cap2"},
			wantImplicit: sets.New[configv1.ClusterVersionCapability]("cap1", "cap2", "cap3"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caps := SetCapabilities(test.config, test.priorEnabled)
			if diff := cmp.Diff(test.wantImplicit, caps.ImplicitlyEnabled); diff != "" {
				t.Errorf("%s: wantImplicit differs from expected:\n%s", test.name, diff)
			}
		})
	}
}

func TestGetCapabilitiesStatus(t *testing.T) {
	tests := []struct {
		name       string
		caps       ClusterCapabilities
		wantStatus configv1.ClusterVersionCapabilitiesStatus
	}{
		{name: "empty capabilities",
			caps: ClusterCapabilities{
				Known:   sets.New[configv1.ClusterVersionCapability](),
				Enabled: sets.New[configv1.ClusterVersionCapability](),
			},
			wantStatus: configv1.ClusterVersionCapabilitiesStatus{
				EnabledCapabilities: []configv1.ClusterVersionCapability{},
				KnownCapabilities:   []configv1.ClusterVersionCapability{},
			},
		},
		{name: "capabilities",
			caps: ClusterCapabilities{
				Known:   sets.New[configv1.ClusterVersionCapability](configv1.ClusterVersionCapabilityOpenShiftSamples),
				Enabled: sets.New[configv1.ClusterVersionCapability](configv1.ClusterVersionCapabilityOpenShiftSamples),
			},
			wantStatus: configv1.ClusterVersionCapabilitiesStatus{
				EnabledCapabilities: []configv1.ClusterVersionCapability{configv1.ClusterVersionCapabilityOpenShiftSamples},
				KnownCapabilities:   []configv1.ClusterVersionCapability{configv1.ClusterVersionCapabilityOpenShiftSamples},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := GetCapabilitiesStatus(test.caps)
			if len(config.KnownCapabilities) != len(test.wantStatus.KnownCapabilities) {
				t.Errorf("Incorrect number of Known keys, wanted: %q. Known returned: %v",
					test.wantStatus.KnownCapabilities, config.KnownCapabilities)
			}
			for _, v := range test.wantStatus.KnownCapabilities {
				vFound := false
				for _, cv := range config.KnownCapabilities {
					if v == cv {
						vFound = true
						break
					}
					if !vFound {
						t.Errorf("Missing Known key %q. Known returned : %v", v, config.KnownCapabilities)
					}
				}
			}
			if len(config.EnabledCapabilities) != len(test.wantStatus.EnabledCapabilities) {
				t.Errorf("Incorrect number of Enabled keys, wanted: %q. Enabled returned: %v",
					test.wantStatus.EnabledCapabilities, config.EnabledCapabilities)
			}
			for _, v := range test.wantStatus.EnabledCapabilities {
				vFound := false
				for _, cv := range config.EnabledCapabilities {
					if v == cv {
						vFound = true
						break
					}
					if !vFound {
						t.Errorf("Missing Enabled key %q. Enabled returned : %v", v, config.EnabledCapabilities)
					}
				}
			}
		})
	}
}

func TestSetFromImplicitlyEnabledCapabilities(t *testing.T) {
	tests := []struct {
		name             string
		implicit         []configv1.ClusterVersionCapability
		capabilities     ClusterCapabilities
		wantCapabilities ClusterCapabilities
	}{
		{name: "implicitly enable capabilities",
			implicit: []configv1.ClusterVersionCapability{
				configv1.ClusterVersionCapability("cap2"),
				configv1.ClusterVersionCapability("cap3"),
				configv1.ClusterVersionCapability("cap4"),
			},
			capabilities: ClusterCapabilities{
				Known:             sets.New[configv1.ClusterVersionCapability]("cap1", "cap2", "cap3", "cap4", "cap5"),
				Enabled:           sets.New[configv1.ClusterVersionCapability]("cap1"),
				ImplicitlyEnabled: sets.New[configv1.ClusterVersionCapability]("cap1"),
			},
			wantCapabilities: ClusterCapabilities{
				Known:             sets.New[configv1.ClusterVersionCapability]("cap1", "cap2", "cap3", "cap4", "cap5"),
				Enabled:           sets.New[configv1.ClusterVersionCapability]("cap1", "cap2", "cap3", "cap4"),
				ImplicitlyEnabled: sets.New[configv1.ClusterVersionCapability]("cap2", "cap3", "cap4"),
			},
		},
		{name: "already enabled capability",
			implicit: []configv1.ClusterVersionCapability{
				configv1.ClusterVersionCapability("cap2"),
			},
			capabilities: ClusterCapabilities{
				Known:   sets.New[configv1.ClusterVersionCapability]("cap1", "cap2"),
				Enabled: sets.New[configv1.ClusterVersionCapability]("cap1", "cap2"),
			},
			wantCapabilities: ClusterCapabilities{
				Known:             sets.New[configv1.ClusterVersionCapability]("cap1", "cap2"),
				Enabled:           sets.New[configv1.ClusterVersionCapability]("cap1", "cap2"),
				ImplicitlyEnabled: sets.New[configv1.ClusterVersionCapability]("cap2"),
			},
		},
		{name: "no implicitly enabled capabilities",
			capabilities: ClusterCapabilities{
				Known:             sets.New[configv1.ClusterVersionCapability]("cap1", "cap2"),
				Enabled:           sets.New[configv1.ClusterVersionCapability]("cap1"),
				ImplicitlyEnabled: sets.New[configv1.ClusterVersionCapability]("cap2"),
			},
			wantCapabilities: ClusterCapabilities{
				Known:   sets.New[configv1.ClusterVersionCapability]("cap1", "cap2"),
				Enabled: sets.New[configv1.ClusterVersionCapability]("cap1"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caps := SetFromImplicitlyEnabledCapabilities(test.implicit, test.capabilities)
			if !reflect.DeepEqual(caps, test.wantCapabilities) {
				t.Fatalf("unexpected: %#v", caps)
			}
		})
	}
}
