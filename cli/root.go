package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand builds the root cobra.Command for libfossil.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "libfossil [command] [flags] [args]",
		Short: "Fossil-compatible repository tool (pure Go)",
	}

	root.PersistentFlags().StringVarP(&Repo, "repo", "R", "", "Path to repository file")
	root.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Verbose output")

	root.AddCommand(newNewCommand())
	root.AddCommand(newCloneCommand())
	root.AddCommand(newServerCommand())
	root.AddCommand(NewCiCommand())
	root.AddCommand(newCoCommand())
	root.AddCommand(newLsCommand())
	root.AddCommand(newTimelineCommand())
	root.AddCommand(newCatCommand())
	root.AddCommand(newInfoCommand())
	root.AddCommand(newHashCommand())
	root.AddCommand(newDeltaCommand())
	root.AddCommand(newConfigCommand())
	root.AddCommand(newQueryCommand())
	root.AddCommand(newVerifyCommand())
	root.AddCommand(newResolveCommand())
	root.AddCommand(newExtractCommand())
	root.AddCommand(newWikiCommand())
	root.AddCommand(newTagCommand())
	root.AddCommand(NewOpenCommand())
	root.AddCommand(newStatusCommand())
	root.AddCommand(newAddCommand())
	root.AddCommand(newRmCommand())
	root.AddCommand(newRenameCommand())
	root.AddCommand(newRevertCommand())
	root.AddCommand(newDiffCommand())
	root.AddCommand(newMergeCommand())
	root.AddCommand(newConflictsCommand())
	root.AddCommand(newMarkResolvedCommand())
	root.AddCommand(newUndoCommand())
	root.AddCommand(newRedoCommand())
	root.AddCommand(newStashCommand())
	root.AddCommand(newBisectCommand())
	root.AddCommand(newAnnotateCommand())
	root.AddCommand(newBlameCommand())
	root.AddCommand(newBranchCommand())
	root.AddCommand(newUVCommand())
	root.AddCommand(newSchemaCommand())
	root.AddCommand(newUserCommand())
	root.AddCommand(newInviteCommand())

	return root
}
