package check

import (
	"context"
	"github.com/chislab/go-fiscobcos/common"
	"github.com/urfave/cli/v2"
)

func Test(ctx *cli.Context) error {
	// 0x694a11351c966ba9102706c5695343d2b9d84e907bc47989deb064058a316881 0xbc3fba53df5282971d81f752f8cd0e2e1f31697e976994bef61f8a844112793c
	_, err := GethCli.TransactionReceipt(context.Background(), common.HexToHash("0x694a11351c966ba9102706c5695343d2b9d84e907bc47989deb064058a316881"))
	if err != nil {
		return err
	}
	return nil
}
