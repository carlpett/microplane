package cmd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/Clever/microplane/clone"
	"github.com/Clever/microplane/initialize"
	"github.com/facebookgo/errgroup"
	"github.com/spf13/cobra"
)

func loadJSON(path string, obj interface{}) error {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, obj)
}

func writeJSON(obj interface{}, path string) error {
	b, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

var cloneCmd = &cobra.Command{
	Use:   "clone [target]",
	Args:  cobra.ExactArgs(1),
	Short: "clone short description",
	Long: `clone
                long
                description`,
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0] // TODO: does ExactArgs(1) above guarantee this will be filled?
		var initOutput initialize.Output
		if err := loadJSON(path.Join(workDir, target, "init.json"), &initOutput); err != nil {
			log.Fatal(err)
		}

		singleRepo, err := cmd.Flags().GetString("repo")
		if err != nil {
			valid := false
			for _, r := range initOutput.Repos {
				if r.Name == singleRepo {
					valid = true
					break
				}
			}
			if !valid {
				log.Fatalf("%s not a targeted repo name", singleRepo) // TODO: showing valid repo names would be helpful
			}
		}

		ctx := context.Background()

		var eg errgroup.Group
		// TODO: limit # of parallel clones
		for _, r := range initOutput.Repos {
			if singleRepo != "" && r.Name != singleRepo {
				continue
			}
			cloneWorkDir := path.Join(workDir, target, r.Name, "clone")
			if err := os.MkdirAll(workDir, 0755); err != nil {
				log.Fatal(err)
			}
			eg.Add(1)
			go func(cloneInput clone.Input) {
				defer eg.Done()
				output, err := clone.Clone(ctx, cloneInput)
				// TODO: should we also write the error? only saving output means "status" command only has Success: true/false to work with
				writeJSON(output, path.Join(cloneInput.WorkDir, "clone.json"))
				if err != nil {
					eg.Error(err)
					return
				}
			}(clone.Input{
				WorkDir: cloneWorkDir,
				GitURL:  r.CloneURL,
			})
		}
		if err := eg.Wait(); err != nil {
			log.Fatal(err)
		}
	},
}
