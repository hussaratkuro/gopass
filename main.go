package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

func main() {
    upperFlag := flag.BoolP("upper", "u", false, "Include uppercase characters")
    lowerFlag := flag.BoolP("lower", "l", false, "Include lowercase characters")
    symbolFlag := flag.BoolP("symbols", "s", false, "Include symbols")
    numberFlag := flag.BoolP("numbers", "n", false, "Include numbers")
    filePathFlag := flag.StringP("file", "f", "", "Save password to a file (requires filename argument)")
    helpFlag := flag.BoolP("help", "h", false, "Display help")

    flag.Usage = func() {
        printUsage()
    }
    flag.Parse()

    if *helpFlag {
        printUsage()
        return
    }

    if flag.NArg() < 1 {
        printUsage()
        return
    }

    lengthArg := flag.Arg(0)
    length, err := strconv.Atoi(lengthArg)
    if err != nil || length <= 0 || length > 50 {
        fmt.Println("Error: Length must be a number between 1 and 50.")
        printUsage()
        return
    }

    var charset string
    if *upperFlag {
        charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    }
    if *lowerFlag {
        charset += "abcdefghijklmnopqrstuvwxyz"
    }
    if *symbolFlag {
        charset += "!@#$%^&*()-_=+[]{}|;:,.<>/?"
    }
    if *numberFlag {
        charset += "0123456789"
    }

    if charset == "" {
        fmt.Println("Error: No character sets selected. Please enable at least one of -u, -l, -s, or -n.")
        printUsage()
        return
    }

    password, err := generatePassword(length, charset)
    if err != nil {
        fmt.Println("Error generating password:", err)
        return
    }

    fmt.Printf("New password: %s\n", password)

    if *filePathFlag != "" {
        filename := *filePathFlag
        if !strings.Contains(filename, ".") {
            filename += ".txt"
        }

        var file *os.File
        if _, err := os.Stat(filename); os.IsNotExist(err) {
            file, err = os.Create(filename)
            if err != nil {
                fmt.Println("Error creating file:", err)
                return
            }
        } else {
            file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
            if err != nil {
                fmt.Println("Error opening file:", err)
                return
            }
            if _, err = file.WriteString("\n\n"); err != nil {
                fmt.Println("Error writing to file:", err)
                return
            }
        }
        defer file.Close()

        if _, err = file.WriteString(password); err != nil {
            fmt.Println("Error writing password to file:", err)
            return
        }
        fmt.Printf("Password saved to %s\n", filename)
    }
}

func generatePassword(length int, charset string) (string, error) {
    password := make([]byte, length)
    for i := range password {
        index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", err
        }
        password[i] = charset[index.Int64()]
    }
    return string(password), nil
}

func printUsage() {
    usage := "   ____ _____  ____  ____ ___________\n" +
        "  / __ `/ __ \\/ __ \\/ __ `/ ___/ ___/\n" +
        " / /_/ / /_/ / /_/ / /_/ (__  |__  ) \n" +
        " \\__, /\\____/ .___/\\__,_/____/____/  \n" +
        "/____/     /_/                       \n" +
        "(Compiled Fri Feb 28 19:49:55 2025)\n\n" +
        "Usage:\n" +
        "  gopass [options] <length>\n\n" +
        "Options:\n" +
        "  -u, --upper       Use uppercase characters\n" +
        "  -l, --lower       Use lowercase characters\n" +
        "  -s, --symbols     Use symbols\n" +
        "  -n, --numbers     Use numbers\n" +
        "  -f, --file <file> Save password to a file (requires filename argument)\n" +
        "  -h, --help        Display this help message\n\n" +
        "Example:\n" +
        "  gopass -ulsn -f ./example.txt 16\n" +
        "  This generates a 16-character password using uppercase, lowercase, symbols, and numbers.\n" +
        "  The password is printed and also appended to \"./example.txt\".\n"
    fmt.Println(usage)
}
