---
title: Bash Autocomplete
sort_key: C
---

You can enable autocompletion in geth just running a bash script (Linux/MacOS) or a powershell script (Windows).

### Linux/MacOS

1.  Create a bash script file with the content below and save as `geth-autocompletion` anywhere in your computer (i.e. `/bin/geth-autocompletion`):

    ```bash
    #! /bin/bash

    : ${PROG:=$(basename ${BASH_SOURCE})}

    _cli_bash_autocomplete() {
    if [[ "${COMP_WORDS[0]}" != "source" ]]; then
        local cur opts base
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        if [[ "$cur" == "-"* ]]; then
        opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} ${cur} --generate-bash-completion )
        else
        opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
        fi
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
    }

    complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete $PROG
    unset PROG
    ```

2.  Open and edit your **startup script** depending on the terminal in use (i.e. `~/.bashrc` or `~/.zshrc`).

3.  Includes this command in the final of your **startup script**:

    ```bash
    # i.e. PROG=geth source /bin/geth-autocompletion
    PROG=geth source /path/to/autocomplete/geth-autocompletion-script
    ```

### Windows

1.  Create a powershell script file with the content below and save as `geth.ps1` anywhere in your computer.

    ```bash
    $fn = $($MyInvocation.MyCommand.Name)
    $name = $fn -replace "(.*)\.ps1$", '$1'
    Register-ArgumentCompleter -Native -CommandName $name -ScriptBlock {
        param($commandName, $wordToComplete, $cursorPosition)
        $other = "$wordToComplete --generate-bash-completion"
            Invoke-Expression $other | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
    }
    ```

2.  Open the PowerShell profile (`code $profile` or `notepad $profile`) and add the line:

    ```bash
    & path/to/autocomplete/geth.ps1
    ```

