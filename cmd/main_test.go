package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	"github.com/argoproj-labs/argocd-notifications/notifiers"
	"github.com/argoproj-labs/argocd-notifications/triggers"
)

func TestMergeConfigTemplate(t *testing.T) {
	cfg := config{
		Templates: []triggers.NotificationTemplate{{
			Name: "foo",
			Notification: notifiers.Notification{
				Body:  "the body",
				Title: "the title",
				Slack: &notifiers.SlackSpecific{
					Attachments: "attachments",
				},
			},
		}},
	}
	merged := cfg.merge(&config{
		Templates: []triggers.NotificationTemplate{{
			Name: "foo",
			Notification: notifiers.Notification{
				Body: "new body",
				Slack: &notifiers.SlackSpecific{
					Blocks: "blocks",
				},
			},
		}, {
			Name: "bar",
			Notification: notifiers.Notification{
				Body:  "the body",
				Title: "the title",
			},
		}},
	})
	assert.ElementsMatch(t, merged.Templates, []triggers.NotificationTemplate{{
		Name: "foo",
		Notification: notifiers.Notification{
			Title: "the title",
			Body:  "new body",
			Slack: &notifiers.SlackSpecific{
				Attachments: "attachments",
				Blocks:      "blocks",
			},
		},
	}, {
		Name: "bar",
		Notification: notifiers.Notification{
			Body:  "the body",
			Title: "the title",
		},
	}})
}

func TestMergeConfigTriggers(t *testing.T) {
	cfg := config{
		Triggers: []triggers.NotificationTrigger{{
			Name:      "foo",
			Template:  "template name",
			Condition: "the condition",
			Enabled:   pointer.BoolPtr(false),
		}},
	}

	merged := cfg.merge(&config{
		Triggers: []triggers.NotificationTrigger{{
			Name:      "foo",
			Condition: "new condition",
			Enabled:   pointer.BoolPtr(true),
		}, {
			Name:      "bar",
			Condition: "the condition",
			Template:  "the template",
			Enabled:   pointer.BoolPtr(true),
		}},
	})

	assert.ElementsMatch(t, merged.Triggers, []triggers.NotificationTrigger{{
		Name:      "foo",
		Template:  "template name",
		Condition: "new condition",
		Enabled:   pointer.BoolPtr(true),
	}, {
		Name:      "bar",
		Condition: "the condition",
		Template:  "the template",
		Enabled:   pointer.BoolPtr(true),
	}})
}

func TestWatchConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: "default",
		},
		Data: map[string]string{
			"config.yaml": `
triggers:
  - name: on-sync-status-unknown
    template: app-sync-status
    enabled: true
templates:
  - name: app-sync-status
    title: updated
    body: updated"`,
		},
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"notifiers.yaml": []byte(`slack: {token: my-token}`),
		},
	}

	triggersMap := make(map[string]triggers.Trigger)
	notifiersMap := make(map[string]notifiers.Notifier)
	clientset := fake.NewSimpleClientset(configMap, secret)
	watchConfig(ctx, clientset, "default", func(t map[string]triggers.Trigger, n map[string]notifiers.Notifier, ctx map[string]string) error {
		triggersMap = t
		notifiersMap = n
		return nil
	})

	assert.Len(t, triggersMap, 1)

	_, ok := triggersMap["on-sync-status-unknown"]
	assert.True(t, ok)

	_, ok = notifiersMap["slack"]
	assert.True(t, ok)
}
