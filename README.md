
# 🐙 Roo 

## Install
```
brew tap outlaw/homebrew-tap
brew install outlaw/homebrew-tap/roo
```

## Usage

### Environment Variables
Roo helps you store your application environment variables in a secure way.

To Store:
```
echo "opensaysme" | roo env set --application my-app --environment production SECRET_ENVIRONMENT_VARIABLE
```

To Retrieve:
```
roo env get --application my-app --environment production SECRET_ENVIRONMENT_VARIABLE
```

Simples!

### Other
```
create
deploy
lockbox
destroy
```
