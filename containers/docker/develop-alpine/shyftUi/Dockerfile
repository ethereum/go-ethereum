FROM node:8.11.3-alpine

# set working directory
RUN echo `cd usr && ls -a`
RUN mkdir -p usr/src/shyftBlockExplorerUI
WORKDIR /usr/src/shyftBlockExplorerUI

# add `/usr/src/app/node_modules/.bin` to $PATH
ENV PATH /usr/src/shyftBlockExplorerUI/node_modules/.bin:$PATH

# install and cache app dependencies
COPY ./shyftBlockExplorerUI/package.json /usr/src/shyftBlockExplorerUI/package.json
RUN npm install
# RUN npm add react-scripts@1.1.1 -g --silent
COPY ./shyftBlockExplorerUI/src /usr/src/shyftBlockExplorerUI/src
COPY ./shyftBlockExplorerUI/scripts /usr/src/shyftBlockExplorerUI/scripts
COPY ./shyftBlockExplorerUI/config /usr/src/shyftBlockExplorerUI/config
COPY ./shyftBlockExplorerUI/public /usr/src/shyftBlockExplorerUI/public

# start app

EXPOSE 3000

CMD ["npm", "start"]
 
