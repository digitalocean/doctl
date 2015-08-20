package commands

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Actions creates the action commands heirarchy.
func Actions() *cobra.Command {
	cmdActions := &cobra.Command{
		Use:   "action",
		Short: "action commands",
		Long:  "action is used to access action commands",
	}

	cmdActionList := &cobra.Command{
		Use:   "list",
		Short: "action list",
		Long:  "list actions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("listing actions")
		},
	}

	cmdActionGet := &cobra.Command{
		Use:   "get",
		Short: "action get",
		Long:  "get action",
		Run: func(cmd *cobra.Command, args []string) {
			client := doit.GetClient()
			id := viper.GetInt(doit.ArgActionID)
			if id < 1 {
				logrus.Fatal("invalid action id")
			}

			a, _, err := client.Actions.Get(id)
			if err != nil {
				logrus.WithField("err", err).Fatal("unable to retrieve action")
			}

			err = doit.DisplayOutput(a)
			if err != nil {
				logrus.WithField("err", err).Fatal("unable to display action")
			}
		},
	}

	cmdActionGet.Flags().Int(doit.ArgActionID, 0, "Action ID")
	viper.BindPFlag(doit.ArgActionID, cmdActionGet.Flags().Lookup(doit.ArgActionID))

	cmdActions.AddCommand(cmdActionList, cmdActionGet)

	return cmdActions
}
