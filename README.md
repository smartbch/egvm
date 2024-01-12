# Enclave-Guarded Virtual Machine

A javascript VM guarded by enclaves.

### Prerequisite
- A linux server that supports Intel SGX
- Install [ego](https://docs.edgeless.systems/ego)

### Install
```bash
git clone https://github.com/smartbch/egvm
cd egvm
go mod download -x
```


### Run
#### keygrantor
1. modify `keygrantor/encalve.json`
    ```json
    {
      "exe": "keygrantor",
      "key": "private.pem",
      "debug": false,
      "heapSize": 512,
      "executableHeap": false,
      "productID": 1,
      "securityVersion": 3,
      "mounts": [
        {
          "source": "<your_path_to_store_private_key>",
          "target": "/data",
          "type": "hostfs",
          "readOnly": false
        },
        {
          "source": "/etc/ssl",
          "target": "/etc/ssl",
          "type": "hostfs",
          "readOnly": true
        }
      ],
      "env": null,
      "files": null
    }
    ```

2. build and run
    ```bash
    cd keygrantor
    ego-go build -o keygrantor ./cmd
    ego sign enclave.json
    ego run keygrantor
    ```

#### egvm
1. build
    ```bash
    cd egvm-invoker
    go build -o egvminvoker ./cmd
    go build -o invokertester ./tester
    ego-go build -o egvmscript ../egvm-script/cmd
    ego sign egvmscript
    ```
   
2. run
   ```bash
   ./egvminvoker
   ```
   
3. test
   ```bash
   ./invokertester -w ./tester/script/write_output.txt
   ```
