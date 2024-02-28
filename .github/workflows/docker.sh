#!/bin/sh

set -ex

image="$ECR_REGISTRY/$ECR_REPOSITORY"
tag=$(git tag --points-at HEAD)

if [ -z "$tag" ]; then
  tag='latest'
fi

echo $image:$tag
docker buildx create --name mybuilder --use --bootstrap || echo 'skip'
docker buildx inspect --bootstrap

dockerfile="./Dockerfile"
docker buildx build \
  --progress plain \
  --build-arg BUILDKIT_INLINE_CACHE=1 \
  --cache-from type=registry,ref=${image}:cache \
  --cache-to type=registry,image-manifest=true,oci-mediatypes=true,ref=${image}:cache,mode=max \
  --compress \
  --push \
  --platform linux/amd64,linux/arm64 \
  -t "$image:$tag" \
  -f "${dockerfile}" .
docker buildx imagetools inspect $image:$tag
echo "image=$image:$tag" >> $GITHUB_OUTPUT
