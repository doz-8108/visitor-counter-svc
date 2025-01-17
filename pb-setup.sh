git rm --cached pb> /dev/null 2>&1

if [ ! -f ".gitmodules" ]; then
    touch .gitmodules
fi

dependencies=("visitor-counter-svc")
if [ ! -d "pb" ]; then
    git submodule add --force "git@github.com:doz-8108/protobufs.git" pb
    cd pb

    folders=$(find . -maxdepth 1 -type d ! -name '.' ! -name '..' | sed 's|^\./||')
    for folder in ${folders[@]}; do
        if [[ ! " ${dependencies[*]} " =~ $folder ]]; then
            rm -rf $folder 
        else
            mv ./$folder/* ./
            rm -rf $folder
        fi
    done
fi