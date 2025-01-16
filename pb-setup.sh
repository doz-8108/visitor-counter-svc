if [ ! -d "pb" ]; then \
    git clone --no-checkout git@github.com:doz-8108/protobufs.git pb && \
    cd pb && \
    git sparse-checkout init && \
    git sparse-checkout set visitor-counter-svc && \
    git checkout main && mv ./visitor-counter-svc/* . && rm -rf ./visitor-counter-svc; \
fi