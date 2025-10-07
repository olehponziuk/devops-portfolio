#!/bin/bash

cp hooks/pre-commit .git/hooks/
cp hooks/pre-push .git/hooks/
cp hooks/post-commit .git/hooks/

chmod +x .git/hooks/pre-commit
chmod +x .git/hooks/pre-push
chmod +x .git/hooks/post-commit
