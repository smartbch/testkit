package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"

	"github.com/smartbch/testkit/bchnode/generator/types"
)

const (
	flagTxid           = "txid"
	flagCcCovenantAddr = "cc-covenant-addr"
	flagAmt            = "amt"
	flagOpRet          = "op-return"
	flagScriptSigHex   = "script-sig-hex"
	flagInTxid         = "in-txid"
	flagInVout         = "in-vout"
)

func main() {
	rootCmd := createRootCmd()
	executor := cli.Executor{Command: rootCmd, Exit: os.Exit}
	err := executor.Execute()
	if err != nil {
		panic(err)
	}
}

func createRootCmd() *cobra.Command {
	cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:   "bchutxomaker",
		Short: "UTXO maker for fakenode",
	}

	rootCmd.AddCommand(transferToSbchCmd())
	rootCmd.AddCommand(redeemByUserCmd())
	rootCmd.AddCommand(convertByOperatorsCmd())
	rootCmd.AddCommand(convertByMonitorsCmd())
	return rootCmd
}

func transferToSbchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make-cc-utxo",
		Short: "make cc-UTXO",
		Example: `bchutxomaker make-cc-utxo \
	--txid=c01ab2bfa4a7f64cf781e886844de836e7b45f2c6150de380cb891045e8353c9 \
	--cc-covenant-addr=6ad3f81523c87aa17f1dfa08271cf57b6277c98e \
	--amt=0.001 \
	--op-return=0xc370743331b37d3c6d0ee798b3918f6561af2c92 \
	--script-sig-hex=483045022100fcf716f6b6cb75be60c1b4f399facc7bc596fdcb521008cb0ccb6d8045a20f6a0220236859c32ee5f7868e6c97657dc35cfd2143b6ec5c5b62eba7006b0f63cc9b00412102d27c31afad03f4a300868165b5aff09babe6bb3fdc14048ecb3e1de1457c4b3e`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			txid := viper.GetString(flagTxid)
			ccCovenantAddr := viper.GetString(flagCcCovenantAddr)
			amt := viper.GetFloat64(flagAmt)
			opRet := viper.GetString(flagOpRet)
			scriptSig := viper.GetString(flagScriptSigHex)

			tx := types.TxInfo{}
			tx.Version = 2
			if txid != "" {
				tx.TxID = txid
				tx.Hash = txid
			}
			if scriptSig != "" {
				tx.VinList = append(tx.VinList, map[string]interface{}{
					"scriptSig": map[string]string{
						"hex": scriptSig, // TODO
					},
				})
			}
			tx.VoutList = append(tx.VoutList, types.Vout{
				Value: amt,
				ScriptPubKey: map[string]interface{}{
					"asm": "OP_HASH160 " + ccCovenantAddr + " OP_EQUAL",
				},
			})
			if opRet != "" {
				tx.VoutList = append(tx.VoutList, types.Vout{
					ScriptPubKey: map[string]interface{}{
						"asm": "OP_RETURN " + hex.EncodeToString([]byte(opRet)),
					},
				})
			}

			data, _ := json.MarshalIndent(tx, "", "  ")
			fmt.Println(string(data))

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String(flagTxid, "", "TXID")
	cmd.Flags().String(flagCcCovenantAddr, "", "P2SH address of cc-covenant")
	cmd.Flags().Float64(flagAmt, 0, "how many BCH to transfer")
	cmd.Flags().String(flagOpRet, "", "sBCH address to be put into OP_RETURN")
	cmd.Flags().String(flagScriptSigHex, "", "scriptSig to find sender address")
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagCcCovenantAddr)
	_ = cmd.MarkFlagRequired(flagAmt)

	return cmd
}

func redeemByUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-cc-utxo",
		Short: "redeem cc-UTXO",
		Example: `bchutxomaker redeem-cc-utxo \
	--txid=c01ab2bfa4a7f64cf781e886844de836e7b45f2c6150de380cb891045e8353c9 \
	--in-txid=4798e7b278130160bc5fdfe1d0f297786c9268a1631ea6a00f531e5f3e798f73 \
	--in-vout=1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			txid := viper.GetString(flagTxid)
			inTxid := viper.GetString(flagInTxid)
			inVout := viper.GetUint(flagInVout)

			tx := types.TxInfo{}
			tx.Version = 2
			if txid != "" {
				tx.TxID = txid
				tx.Hash = txid
			}
			tx.VinList = append(tx.VinList, map[string]interface{}{
				"txid": inTxid,
				"vout": inVout,
			})
			tx.VoutList = append(tx.VoutList, types.Vout{
				ScriptPubKey: map[string]interface{}{
					"asm": "OP_DUP OP_HASH160 f1c075a01882ae0972f95d3a4177c86c852b7d91 OP_EQUALVERIFY OP_CHECKSIG",
				},
			})

			data, _ := json.MarshalIndent(tx, "", "  ")
			fmt.Println(string(data))

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String(flagTxid, "", "tx TXID")
	cmd.Flags().String(flagInTxid, "", "input TXID")
	cmd.Flags().Uint(flagInVout, 0, "input vout")
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagInTxid)
	_ = cmd.MarkFlagRequired(flagInVout)
	return cmd
}

func convertByOperatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-by-operators",
		Short: "convert cc-UTXO by operators",
		Example: `go run github.com/smartbch/testkit/bchutxomaker convert-by-operators \
	--txid=c01ab2bfa4a7f64cf781e886844de836e7b45f2c6150de380cb891045e8353c9 \
	--in-txid=4798e7b278130160bc5fdfe1d0f297786c9268a1631ea6a00f531e5f3e798f73 \
	--in-vout=1 \
	--amt=0.001 \
	--cc-covenant-addr=6ad3f81523c87aa17f1dfa08271cf57b6277c98e`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			txid := viper.GetString(flagTxid)
			ccCovenantAddr := viper.GetString(flagCcCovenantAddr)
			amt := viper.GetFloat64(flagAmt)
			inTxid := viper.GetString(flagInTxid)
			inVout := viper.GetUint(flagInVout)

			tx := types.TxInfo{}
			tx.Version = 2
			if txid != "" {
				tx.TxID = txid
				tx.Hash = txid
			}
			tx.VinList = append(tx.VinList, map[string]interface{}{
				"txid": inTxid,
				"vout": inVout,
			})
			tx.VoutList = append(tx.VoutList, types.Vout{
				Value: amt,
				ScriptPubKey: map[string]interface{}{
					"asm": "OP_HASH160 " + ccCovenantAddr + " OP_EQUAL",
				},
			})

			data, _ := json.MarshalIndent(tx, "", "  ")
			fmt.Println(string(data))

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String(flagTxid, "", "tx TXID")
	cmd.Flags().String(flagCcCovenantAddr, "", "new P2SH address of cc-covenant")
	cmd.Flags().Float64(flagAmt, 0, "value of UTXO")
	cmd.Flags().String(flagInTxid, "", "input TXID")
	cmd.Flags().Uint(flagInVout, 0, "input vout")
	_ = cmd.MarkFlagRequired(flagTxid)
	_ = cmd.MarkFlagRequired(flagCcCovenantAddr)
	_ = cmd.MarkFlagRequired(flagAmt)
	_ = cmd.MarkFlagRequired(flagInTxid)
	_ = cmd.MarkFlagRequired(flagInVout)
	return cmd
}

func convertByMonitorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-by-monitors",
		Short: "convert cc-UTXO by monitors",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			txid := viper.GetString(flagTxid)

			tx := types.TxInfo{}
			tx.Version = 2
			if txid != "" {
				tx.TxID = txid
				tx.Hash = txid
			}

			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String(flagTxid, "", "tx TXID")
	_ = cmd.MarkFlagRequired(flagTxid)
	return cmd
}
