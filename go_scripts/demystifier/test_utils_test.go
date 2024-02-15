package demystifier

import "testing"

func TestGenerateLogURL(t *testing.T) {
	type args struct {
		originalURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test with pull-ci-openshift-oadp-operator-master-4.12-e2e-test-azure",
			args: args{
				originalURL: "https://prow.ci.openshift.org/view/gs/test-platform-results/pr-logs/pull/openshift_oadp-operator/1330/pull-ci-openshift-oadp-operator-master-4.12-e2e-test-azure/1757841602983759872",
			},
			want: "https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/test-platform-results/pr-logs/pull/openshift_oadp-operator/1330/pull-ci-openshift-oadp-operator-master-4.12-e2e-test-azure/1757841602983759872/artifacts/e2e-test-azure/e2e/build-log.txt",
		},
		{
			name: "Test with pull-ci-openshift-oadp-operator-master-4.14-e2e-test-aws",
			args: args{
				originalURL: "https://prow.ci.openshift.org/view/gs/test-platform-results/pr-logs/pull/openshift_oadp-operator/1330/pull-ci-openshift-oadp-operator-master-4.14-e2e-test-aws/1757841603164114944",
			},
			want: "https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/test-platform-results/pr-logs/pull/openshift_oadp-operator/1330/pull-ci-openshift-oadp-operator-master-4.14-e2e-test-aws/1757841603164114944/artifacts/e2e-test-aws/e2e/build-log.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateLogURL(tt.args.originalURL); got != tt.want {
				t.Errorf("GenerateLogURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
