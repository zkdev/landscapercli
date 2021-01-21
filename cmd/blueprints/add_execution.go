// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/gardener/landscaper/pkg/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscapercli/pkg/logger"
)

type addExecutionOptions struct {
	// blueprintPath is the path to the directory containing the definition.
	blueprintPath string

	// name of the deploy execution
	name string
}

// NewCreateCommand creates a new blueprint command to create a blueprint
func NewAddExecutionCommand(ctx context.Context) *cobra.Command {
	opts := &addExecutionOptions{}
	cmd := &cobra.Command{
		Use:     "execution [path to Blueprint directory] [name]",
		Args:    cobra.ExactArgs(2),
		Example: "landscaper-cli blueprints add execution path/to/blueprint/directory default",
		Short:   "command to add a deploy execution skeleton to the blueprint in the specified directory",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.run(ctx, logger.Log); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			fmt.Printf("Successfully added deploy execution %s\n", opts.name)
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *addExecutionOptions) Complete(args []string) error {
	o.blueprintPath = args[0]
	o.name = args[1]
	return nil
}

func (o *addExecutionOptions) AddFlags(fs *pflag.FlagSet) {
}

func (o *addExecutionOptions) run(ctx context.Context, log logr.Logger) error {
	blueprint, err := o.readBlueprint()
	if err != nil {
		return err
	}

	if o.existsExecution(blueprint) {
		return fmt.Errorf("The blueprint already contains a deploy execution %s\n", o.name)
	}

	exists, err := o.existsExecutionFile()
	if err != nil {
		return err
	}

	if !exists {
		err = o.createExecutionFile()
		if err != nil {
			return err
		}
	}

	o.addExecution(blueprint)

	return o.writeBlueprint(blueprint)
}

func (o *addExecutionOptions) readBlueprint() (*core.Blueprint, error) {
	data, err := ioutil.ReadFile(filepath.Join(o.blueprintPath, lsv1alpha1.BlueprintFileName))
	if err != nil {
		return nil, err
	}

	blueprint := &core.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, blueprint); err != nil {
		return nil, err
	}

	return blueprint, nil
}

func (o *addExecutionOptions) writeBlueprint(blueprint *core.Blueprint) error {
	data, err := yaml.Marshal(blueprint)
	if err != nil {
		return err
	}

	blueprintFilePath := filepath.Join(o.blueprintPath, blueprintFilename)
	f, err := os.Create(blueprintFilePath)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (o *addExecutionOptions) existsExecution(blueprint *core.Blueprint) bool {
	for i := range blueprint.DeployExecutions {
		execution := &blueprint.DeployExecutions[i]
		if execution.Name == o.name {
			return true
		}
	}

	return false
}

func (o *addExecutionOptions) addExecution(blueprint *core.Blueprint) {
	blueprint.DeployExecutions = append(blueprint.DeployExecutions, core.TemplateExecutor{
		Name: o.name,
		Type: "GoTemplate",
		File: "/" + o.getExecutionFileName(),
	})
}

func (o *addExecutionOptions) existsExecutionFile() (bool, error) {
	fileInfo, err := os.Stat(o.getExecutionFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if fileInfo.IsDir() {
		return true, fmt.Errorf("There already exists a directory %s\n", o.getExecutionFileName())
	}

	return true, nil
}

func (o *addExecutionOptions) createExecutionFile() error {
	f, err := os.Create(o.getExecutionFilePath())
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString("deployItems: []\n")

	return err
}

func (o *addExecutionOptions) getExecutionFilePath() string {
	return filepath.Join(o.blueprintPath, o.getExecutionFileName())
}

func (o *addExecutionOptions) getExecutionFileName() string {
	return o.name + "DeployExecution.yaml"
}
