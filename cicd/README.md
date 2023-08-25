# Docker Command

## Docker Build
```
docker build  -f cicd/Dockerfile .
```
## Docker Run
```
docker run -it -e NETWORK=devnet -e PRIVATE_KEYS=$KEY $IMAGE
``