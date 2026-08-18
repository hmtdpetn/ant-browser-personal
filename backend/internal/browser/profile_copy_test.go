package browser

import "testing"

func TestNormalizeCopyAutomationTargetsMapsLegacyTargetsToSeed(t *testing.T) {
	targets, err := normalizeCopyAutomationTargets([]string{"render", "fonts", "devices"})
	if err != nil {
		t.Fatalf("normalizeCopyAutomationTargets returned error: %v", err)
	}
	if len(targets) != 1 || targets[0] != copyAutomationTargetSeed {
		t.Fatalf("targets = %#v, want seed alias", targets)
	}
}

func TestBuildAutoFingerprintArgsKeepsLegacyFineGrainedArgs(t *testing.T) {
	sourceArgs := []string{
		"--fingerprint=111",
		"--fingerprint-canvas-noise=true",
		"--fingerprint-webgl-vendor=Intel",
	}
	defaultArgs := []string{"--fingerprint=222"}

	got := buildAutoFingerprintArgs(sourceArgs, defaultArgs, []string{copyAutomationTargetSeed})
	want := []string{"--fingerprint=222", "--fingerprint-canvas-noise=true", "--fingerprint-webgl-vendor=Intel"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got[%d] = %q, want %q; got=%#v", index, got[index], want[index], got)
		}
	}
}
