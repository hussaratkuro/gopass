## Usage
gopass [options] <length>

## Options:
  -u, --upper       Use uppercase characters<br>
  -l, --lower       Use lowercase characters<br>
  -s, --symbols     Use symbols<br>
  -n, --numbers     Use numbers<br>
  -f, --file <file> Save password to a file (requires filename argument)<br>
  -h, --help        Display this help message<br>

## Example:
  gopass -ulsn -f ./example.txt 16<br>
  This generates a 16-character password using uppercase, lowercase, symbols, and numbers.<br>
  The password is printed and also appended to "./example.txt".<br>
