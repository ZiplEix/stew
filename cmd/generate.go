package cmd

import (
	"fmt"
	"os"

	"github.com/ZiplEix/stew/internal/generator"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Automatically generate the router from the pages directory",
	Run: func(cmd *cobra.Command, args []string) {
		moduleName, err := generator.GetModuleName()
		if err != nil {
			fmt.Printf("❌ Erreur : Impossible de lire go.mod : %v\n", err)
			os.Exit(1)
		}

		scanner := generator.NewScanner("pages", moduleName)

		fmt.Println("🔍 Scanning pages...")
		tree, err := scanner.Scan()
		if err != nil {
			fmt.Printf("❌ Erreur lors du scan : %v\n", err)
			os.Exit(1)
		}

		writer := generator.NewWriter(tree)

		outputFile := "stew_router_gen.go"
		fmt.Printf("🏗️  Generating %s...\n", outputFile)

		if err := writer.Generate(outputFile); err != nil {
			fmt.Printf("❌ Erreur lors de la génération : %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Stew Router généré avec succès !")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
