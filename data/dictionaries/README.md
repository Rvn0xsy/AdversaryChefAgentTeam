# Fuzzing Dictionaries

## Structure

| Directory | Content | Source |
|-----------|---------|--------|
| `dir/` | Directory brute-force wordlists | SecLists/Discovery/Web-Content |
| `api/` | API path wordlists | Custom — common API naming patterns |
| `param/` | Parameter name wordlists | SecLists + custom |
| `pass/` | Password wordlists | SecLists/Passwords/Common-Credentials |

## Updating Dictionaries

Copy files from [SecLists](https://github.com/danielmiessler/SecLists) or add custom entries:

```bash
# Update directory dictionary from SecLists
curl -o dir/common.txt https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/common.txt

# Add custom API paths
echo "/api/internal/users" >> api/common.txt
```

## Custom Entries

Add one entry per line. Blank lines and # comments are ignored by fuzzing tools. Custom entries persist across SecLists updates (they're in separate files).
