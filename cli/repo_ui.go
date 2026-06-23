package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/danmestas/libfossil/internal/webui"
	"github.com/spf13/cobra"
)

func newUICommand() *cobra.Command {
	var addr string
	var noBrowser bool
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Start web interface and open browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			url := fmt.Sprintf("http://%s", addr)
			fmt.Fprintf(cmd.OutOrStdout(), "Serving %s on %s\n", r.Path(), url)

			if !noBrowser {
				if err := openBrowser(url); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not open browser: %v\n", err)
				}
			}

			return webui.ServeBlocks(ctx, addr, r)
		},
	}
	cmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:8080", "Listen address")
	cmd.Flags().BoolVarP(&noBrowser, "no-browser", "n", false, "Don't open a browser")
	return cmd
}

// openBrowser opens the given URL in the system default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
