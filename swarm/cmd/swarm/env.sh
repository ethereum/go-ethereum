export GOPATH=~/go
export PATH=~/bin:$GOPATH/bin:$PATH

if [ -f ~/.bash_aliases ]; then
    . ~/.bash_aliases
fi

export NVM_DIR="/home/ubuntu/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"  # This loads nvm