package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testUptimeAlert = do.UptimeAlert{
		UptimeAlert: &godo.UptimeAlert{
			ID:         "00000000-0000-4000-8000-000000000000",
			Name:       "Test Alert",
			Type:       "latency",
			Comparison: "greater_than",
			Period:     "2m",
			Notifications: &godo.Notifications{
				Email: []string{"test@example.com"},
				Slack: []godo.SlackDetails{
					{URL: "https://hooks.slack.com/services/T1234567/AAAAAAAAA/ZZZZZZ", Channel: "#alerts-test"},
				},
			},
		},
	}
	testUptimeAlertList = []do.UptimeAlert{
		testUptimeAlert,
	}
)

func Test_uptimeAlertCmd(t *testing.T) {
	cmd := uptimeAlertCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "update", "delete")
}

func TestRunUptimeAlertsCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cuar := godo.CreateUptimeAlertRequest{
			Name:          testUptimeAlert.Name,
			Type:          testUptimeAlert.Type,
			Comparison:    testUptimeAlert.Comparison,
			Period:        testUptimeAlert.Period,
			Notifications: testUptimeAlert.Notifications,
		}
		tm.uptimeAlerts.EXPECT().Create("00000000-0000-4000-8000-000000000000", &cuar).Return(&testUptimeAlert, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		config.Doit.Set(config.NS, doctl.ArgUptimeAlertName, "Test Alert")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertType, "latency")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertComparison, "greater_than")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertPeriod, "2m")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertEmails, testUptimeAlert.Notifications.Email)
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackURLs, []string{testUptimeAlert.Notifications.Slack[0].URL})
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackChannels, []string{testUptimeAlert.Notifications.Slack[0].Channel})

		err := RunUptimeAlertsCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunUptimeAlertsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeAlerts.EXPECT().List("00000000-0000-4000-8000-000000000000").Return(testUptimeAlertList, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunUptimeAlertsList(config)
		assert.NoError(t, err)
	})
}

func TestRunUptimeAlertsUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cuar := godo.UpdateUptimeAlertRequest{
			Name:          testUptimeAlert.Name,
			Type:          testUptimeAlert.Type,
			Comparison:    testUptimeAlert.Comparison,
			Period:        testUptimeAlert.Period,
			Notifications: testUptimeAlert.Notifications,
		}
		tm.uptimeAlerts.EXPECT().Update("00000000-0000-4000-8000-000000000000", "00000000-0000-4000-8000-000000000001", &cuar).Return(&testUptimeAlert, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000001")

		config.Doit.Set(config.NS, doctl.ArgUptimeAlertName, "Test Alert")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertType, "latency")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertComparison, "greater_than")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertPeriod, "2m")
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertEmails, testUptimeAlert.Notifications.Email)
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackURLs, []string{testUptimeAlert.Notifications.Slack[0].URL})
		config.Doit.Set(config.NS, doctl.ArgUptimeAlertSlackChannels, []string{testUptimeAlert.Notifications.Slack[0].Channel})

		err := RunUptimeAlertsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunUptimeAlertsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeAlerts.EXPECT().Get("00000000-0000-4000-8000-000000000000", "00000000-0000-4000-8000-000000000001").Return(&testUptimeAlert, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000001")

		err := RunUptimeAlertsGet(config)
		assert.NoError(t, err)
	})
}

func TestRunUptimeAlertsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeAlerts.EXPECT().Delete("00000000-0000-4000-8000-000000000000", "00000000-0000-4000-8000-000000000001").Return(nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000001")

		err := RunUptimeAlertsDelete(config)
		assert.NoError(t, err)
	})
}
