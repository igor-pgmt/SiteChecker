# SiteChecker

This tool gets spywords's result of google and yandex parameters from a CSV list of web-sites.
## Usage

##### You can run this program with next flags.
- Input file name [REQUIRED]:

  -fileInput [string]
  
- Column with http:// web addresses [REQUIRED] (default 0):

  -www [int]
  
- Resulting file name (default "result.csv"):

  -fileResult [string]
  
- First line for file to grab data:

  -firstLine [int]

- Amount of simultaneously jobs:

  -threads [int]
  
- Help command to show this info:

  -help

## Examples
```sh
./SiteChecker -fileInput input.csv -www 2
```
```sh
./SiteChecker -fileInput input.csv -www 1 -firstLine 1 -fileResult output.csv -threads 8
```