# Usage
   ____ _____  ____  ____ ___________
  / __ `/ __ \/ __ \/ __ `/ ___/ ___/
 / /_/ / /_/ / /_/ / /_/ (__  |__  ) 
 \__, /\____/ .___/\__,_/____/____/  
/____/     /_/                       
(Compiled Fri Feb 28 19:43:10 2025)

Usage:
  gopass [options] <length>

Options:
  -u, --upper       Use uppercase characters
  -l, --lower       Use lowercase characters
  -s, --symbols     Use symbols
  -n, --numbers     Use numbers
  -f, --file <file> Save password to a file (requires filename argument)
  -h, --help        Display this help message

Example:
  gopass -ulsn -f ./example.txt 16
  This generates a 16-character password using uppercase, lowercase, symbols, and numbers.
  The password is printed and also appended to "./example.txt".