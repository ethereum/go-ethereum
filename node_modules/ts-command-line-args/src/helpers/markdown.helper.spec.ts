/* eslint-disable no-useless-escape */
import {
    usageGuideInfo as exampleConfigGuideInfo,
    ICopyFilesArguments,
    typicalAppWithGroupsInfo,
    exampleSections,
} from '../example/configs';
import { usageGuideInfo as writeMarkdownGuideInfo } from '../write-markdown.constants';
import { createUsageGuide } from './markdown.helper';
import { UsageGuideConfig } from '../contracts';

describe('markdown-helper', () => {
    it('should generate a simple usage guide with no additional sections and no alias column', () => {
        const info: UsageGuideConfig<ICopyFilesArguments> = {
            arguments: { ...exampleConfigGuideInfo.arguments, copyFiles: Boolean },
        };

        const usageGuide = createUsageGuide(info);

        expect(usageGuide).toEqual(`
## Options

| Argument | Type |
|-|-|
| **sourcePath** | string |
| **targetPath** | string |
| **copyFiles** | boolean |
| **resetPermissions** | boolean |
| **filter** | string |
| **excludePaths** | string[] |
`);
    });

    it('should generate a simple usage guide with typeLabel modifiers', () => {
        const info: UsageGuideConfig<ICopyFilesArguments> = {
            arguments: { ...exampleConfigGuideInfo.arguments, copyFiles: Boolean },
            parseOptions: { displayOptionalAndDefault: true },
        };

        const usageGuide = createUsageGuide(info);

        expect(usageGuide).toEqual(`
## Options

| Argument | Type |
|-|-|
| **sourcePath** | string (D) |
| **targetPath** | string |
| **copyFiles** | boolean |
| **resetPermissions** | boolean |
| **filter** | string (O) |
| **excludePaths** | string[] (O) |
`);
    });

    it('should generate a simple usage guide with typeLabel modifiers and footer', () => {
        const info: UsageGuideConfig<ICopyFilesArguments> = {
            arguments: { ...exampleConfigGuideInfo.arguments, copyFiles: Boolean },
            parseOptions: { addOptionalDefaultExplanatoryFooter: true, displayOptionalAndDefault: true },
        };

        const usageGuide = createUsageGuide(info);

        expect(usageGuide).toEqual(`
## Options

| Argument | Type |
|-|-|
| **sourcePath** | string (D) |
| **targetPath** | string |
| **copyFiles** | boolean |
| **resetPermissions** | boolean |
| **filter** | string (O) |
| **excludePaths** | string[] (O) |
(O) = optional, (D) = default option
`);
    });

    it('should generate a simple usage guide with no additional sections', () => {
        const usageGuide = createUsageGuide(exampleConfigGuideInfo);

        expect(usageGuide).toEqual(`
## Options

| Argument | Alias | Type | Description |
|-|-|-|-|
| **sourcePath** | | string | |
| **targetPath** | | string | |
| **copyFiles** | **c** | **file[]** | **bold text** *italic text* ***bold italic text*** |
| **resetPermissions** | | boolean | |
| **filter** | | string | |
| **excludePaths** | | string[] | |
`);
    });

    it('should generate a usage guide with sections', () => {
        const usageGuide = createUsageGuide(writeMarkdownGuideInfo);

        expect(usageGuide).toEqual(`
## Markdown Generation

A markdown version of the usage guide can be generated and inserted into an existing markdown document.  
Markers in the document describe where the content should be inserted, existing content betweeen the markers is overwritten.



\`write-markdown -m README.MD -j usageGuideConstants.js\`


### write-markdown cli options

| Argument | Alias | Type | Description |
|-|-|-|-|
| **markdownPath** | **m** | string | The file to write to. Without replacement markers the whole file content will be replaced. Path can be absolute or relative. |
| **replaceBelow** | | string | A marker in the file to replace text below. |
| **replaceAbove** | | string | A marker in the file to replace text above. |
| **insertCodeBelow** | | string | A marker in the file to insert code below. File path to insert must be added at the end of the line and optionally codeComment flag: 'insertToken file="path/toFile.md" codeComment="ts"' |
| **insertCodeAbove** | | string | A marker in the file to insert code above. |
| **copyCodeBelow** | | string | A marker in the file being inserted to say only copy code below this line |
| **copyCodeAbove** | | string | A marker in the file being inserted to say only copy code above this line |
| **jsFile** | **j** | string[] | jsFile to 'require' that has an export with the 'UsageGuideConfig' export. Multiple files can be specified. |
| **configImportName** | **c** | string[] | Export name of the 'UsageGuideConfig' object. Defaults to 'usageGuideInfo'. Multiple exports can be specified. |
| **verify** | **v** | boolean | Verify the markdown file. Does not update the file but returns a non zero exit code if the markdown file is not correct. Useful for a pre-publish script. |
| **configFile** | **f** | string | Optional config file to load config from. package.json can be used if jsonPath specified as well |
| **jsonPath** | **p** | string | Used in conjunction with 'configFile'. The path within the config file to load the config from. For example: 'configs.writeMarkdown' |
| **verifyMessage** | | string | Optional message that is printed when markdown verification fails. Use '{fileName}' to refer to the file being processed. |
| **removeDoubleBlankLines** | | boolean | When replacing content removes any more than a single blank line |
| **skipFooter** | | boolean | Does not add the 'Markdown Generated by...' footer to the end of the markdown |
| **help** | **h** | boolean | Show this usage guide. |


### Default Replacement Markers

replaceBelow defaults to:  
  
\`\`\`  
'[//]: ####ts-command-line-args_write-markdown_replaceBelow'  
\`\`\`  
  
replaceAbove defaults to:  
  
\`\`\`  
'[//]: ####ts-command-line-args_write-markdown_replaceAbove'  
\`\`\`  
  
insertCodeBelow defaults to:  
  
\`\`\`  
'[//]: # (ts-command-line-args_write-markdown_insertCodeBelow'  
\`\`\`  
  
insertCodeAbove defaults to:  
  
\`\`\`  
'[//]: # (ts-command-line-args_write-markdown_insertCodeAbove)'  
\`\`\`  
  
copyCodeBelow defaults to:  
  
\`\`\`  
'// ts-command-line-args_write-markdown_copyCodeBelow'  
\`\`\`  
  
copyCodeAbove defaults to:  
  
\`\`\`  
'// ts-command-line-args_write-markdown_copyCodeAbove'  
\`\`\`  

`);
    });

    it('should generate a usage guide with option groups', () => {
        const usageGuide = createUsageGuide(typicalAppWithGroupsInfo);

        expect(usageGuide).toEqual(`
# A typical app

Generates something *very* important.


## Main options

| Argument | Alias | Type | Description |
|-|-|-|-|
| **help** | **h** | boolean | Display this usage guide. |
| **src** | | file ... | Default Option. The input files to process |
| **timeout** | **t** | ms | Defaults to "1000". Timeout value in ms |


## Misc

| Argument | Type | Description |
|-|-|-|
| **plugin** | string | Optional. A plugin path |
`);
    });

    it('should generate a usage guide with table of examples', () => {
        const usageGuide = createUsageGuide(exampleSections);

        expect(usageGuide).toEqual(`
# A typical app

Generates something *very* important.


# both




# markdown




# Synopsis

$ example [**--timeout** ms] **--src** file ...
$ example **--help**


## Options

| Argument | Alias | Type | Description |
|-|-|-|-|
| **help** | **h** | boolean | Display this usage guide. |
| **src** | | file ... | The input files to process |
| **timeout** | **t** | ms | Timeout value in ms |
| **plugin** | | string | A plugin path |


# Examples


| Description | Example |
|-|-|
| 1. A concise example.  | $ example -t 100 lib/*.js |
| 2. A long example.  | $ example --timeout 100 --src lib/*.js |
| 3. This example will scan space for unknown things. Take cure when scanning space, it could take some time.  | $ example --src galaxy1.facts galaxy1.facts galaxy2.facts galaxy3.facts galaxy4.facts galaxy5.facts |


# both




# markdown





Project home: https://github.com/me/example
`);
    });

    it('should generate a usage guide with json example', () => {
        const typicalAppWithJSON: UsageGuideConfig<Record<string, string>> = {
            arguments: {},
            parseOptions: {
                headerContentSections: [
                    {
                        header: 'A typical app',
                        content: `Generates something {italic very} important.
Some Json:

{code.json
\\{
    "dependencies": \\{
        "someDependency: "0.2.1",
    \\},
    "peerDependencies": \\{
        "someDependency: "0.2.1",
    \\}
\\}
}`,
                    },
                ],
            },
        };
        const usageGuide = createUsageGuide(typicalAppWithJSON);

        expect(usageGuide).toEqual(`
# A typical app

Generates something *very* important.  
Some Json:  
  
  
\`\`\`json  
{  
    "dependencies": {  
        "someDependency: "0.2.1",  
    },  
    "peerDependencies": {  
        "someDependency: "0.2.1",  
    }  
}  
  
\`\`\`  

`);
    });
});
