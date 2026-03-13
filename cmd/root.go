/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bfc file.bf",
	Short: "",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: runCompile,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("output", "o", "a.out", "Output binary name (default: a.out)")
}

func runCompile(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(1)
	}

	outputPath := cmd.Flag("output").Value.String()

	inputPath := args[0]

	_, err := os.Stat(inputPath)
	cobra.CheckErr(err)

	source, err := os.ReadFile(inputPath)
	cobra.CheckErr(err)

	assembly := compile(source)

	err = buildBinary(assembly, outputPath)
	cobra.CheckErr(err)
}

// todo optimizations
// [-] should generate `mov byte [rbx], 0` to reset the tape
// duplicate instructions can be consolidated
func compile(source []byte) string {
	var assembly strings.Builder
	var stack []int
	var labelCounter int

	assembly.WriteString(`section .bss
    ; Reserve 30,000 bytes for the Brainfuck "tape"
    tape: resb 30000

section .text
    global _start

_start:
    ; Initialize the Data Pointer (RBX) to the start of our tape
    mov rbx, tape

_main_logic:
`)

	for _, b := range source {
		switch b {
		case '+':
			assembly.WriteString("  inc byte [rbx]\n")
		case '-':
			assembly.WriteString("  dec byte [rbx]\n")
		case '>':
			assembly.WriteString("  inc rbx\n")
		case '<':
			assembly.WriteString("  dec rbx\n")
		case '.':
			assembly.WriteString("  call print_char\n")
		case ',':
			assembly.WriteString("  call read_char\n")
		case '[':
			labelCounter++
			stack = append(stack, labelCounter)
			labelName := "L_START_" + fmt.Sprint(labelCounter)

			assembly.WriteString(labelName + ":\n")
			assembly.WriteString(" cmp byte [rbx], 0\n")
			assembly.WriteString(" je " + "L_END_" + fmt.Sprint(labelCounter) + "\n")
		case ']':
			var pop int
			pop, stack = stack[len(stack)-1], stack[:len(stack)-1]
			labelName := "L_START_" + fmt.Sprint(pop)

			assembly.WriteString(" jmp " + labelName + "\n")
			assembly.WriteString("L_END_" + fmt.Sprint(pop) + ":\n")
		}
	}

	assembly.WriteString(`
_exit:
    ; System call for 'exit' (sys_exit = 60)
    mov rax, 60
    xor rdi, rdi        ; Return code 0
    syscall

print_char:							; . operator
    mov rax, 1          ; sys_write
    mov rdi, 1          ; stdout
    mov rsi, rbx        ; pointer to the char
    mov rdx, 1          ; length 1 byte
    syscall
    ret

read_char: 							; , operator
    mov rax, 0          ; sys_read
    mov rdi, 0          ; stdin
    mov rsi, rbx        ; buffer to store char
    mov rdx, 1          ; length 1 byte
    syscall
    ret
	`)

	return assembly.String()
}

func buildBinary(asmContent string, outputName string) error {
	file, err := os.CreateTemp("", "temp.asm")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(asmContent))
	if err != nil {
		return err
	}
	defer os.Remove(file.Name()) // Cleanup

	// assemble
	nasmCmd := exec.Command("nasm", "-f", "elf64", file.Name(), "-o", "temp.o")
	if err := nasmCmd.Run(); err != nil {
		return fmt.Errorf("NASM error: %v", err)
	}
	defer os.Remove("temp.o")

	// linking
	ldCmd := exec.Command("ld", "temp.o", "-o", outputName)
	if err := ldCmd.Run(); err != nil {
		return fmt.Errorf("LD error: %v", err)
	}

	return nil
}
