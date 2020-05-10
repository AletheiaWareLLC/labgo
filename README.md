labgo
=====

This is a Go implementation of Lab.

Build
=====

    $ go build

Usage
=====

Navigate to, or create, a git repository

    $ git clone ...
    $ cd foobar

Create a new experiment

    $ lab create .
    Created a713df2996f5

Make changes and invite others to collaborate.

Save the experiment back to the file system and commit to git

    $ lab save a713df2996f5 .
    $ git status
    $ git diff
    $ git commit -am "FooBar"

Open existing experiment

    $ lab open a713df2996f5

Open remote experiment locally

    $ lab open a713df2996f5 123.45.67.89

Finally, remove unused experiments using clean

    $ lab clean a713df2996f5
