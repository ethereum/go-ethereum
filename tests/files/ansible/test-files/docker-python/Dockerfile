FROM sveneh/pyethereum-base

RUN git clone --branch master https://github.com/ethereum/pyethereum.git

RUN cd pyethereum && curl https://bootstrap.pypa.io/bootstrap-buildout.py | python

RUN cd pyethereum && bin/buildout

#default port for incoming requests
EXPOSE 30303

WORKDIR /pyethereum/bin 

ENTRYPOINT ["./python", "../tests/test_vm.py"]
