FROM golang:1.26.5-trixie@sha256:f1a132429b98724a904e9b3bdbaed399d8f923203c3e5170e6def66d0a7cc04c AS tools
RUN printf 'deb [check-valid-until=no] https://snapshot.debian.org/archive/debian/20260901T000000Z/ trixie main\ndeb [check-valid-until=no] https://snapshot.debian.org/archive/debian-security/20260901T000000Z/ trixie-security main\n' > /etc/apt/sources.list \
 && find /etc/apt/sources.list.d -type f -delete \
 && apt-get update \
 && apt-get install --no-install-recommends -y cmake ninja-build curl xz-utils libjpeg62-turbo-dev libpng-dev libfontconfig-dev libfreetype-dev liblcms2-dev libtiff-dev libssl-dev zlib1g-dev libzstd-dev libopenjp2-7-dev fonts-dejavu-core
WORKDIR /build
RUN curl --fail --location --proto '=https' --tlsv1.2 -o qpdf.tar.gz https://github.com/qpdf/qpdf/releases/download/v12.4.1/qpdf-12.4.1.tar.gz \
 && printf 'f045aa277be2356ff53a89a8622945958291177d2483afc20ede7c8a8cd3873c  qpdf.tar.gz\n' | sha256sum --check \
 && tar -xzf qpdf.tar.gz \
 && cmake -S qpdf-12.4.1 -B qpdf-build -G Ninja -DCMAKE_BUILD_TYPE=Release -DBUILD_STATIC_LIBS=OFF -DBUILD_DOC=OFF -DBUILD_TESTING=OFF -DREQUIRE_CRYPTO_OPENSSL=ON -DDEFAULT_CRYPTO=openssl \
 && cmake --build qpdf-build --parallel 2 \
 && cmake --install qpdf-build
RUN curl --fail --location --proto '=https' --tlsv1.2 -o poppler.tar.xz https://poppler.freedesktop.org/poppler-26.09.0.tar.xz \
 && printf '8059eadb6805340768f138c465b57f8164c92b4a0773c37ef031ea6c0d987b2e  poppler.tar.xz\n' | sha256sum --check \
 && tar -xf poppler.tar.xz \
 && cmake -S poppler-26.09.0 -B poppler-build -G Ninja -DCMAKE_BUILD_TYPE=Release -DENABLE_QT5=OFF -DENABLE_QT6=OFF -DENABLE_GLIB=OFF -DENABLE_CPP=OFF -DENABLE_LIBCURL=OFF -DENABLE_NSS3=OFF -DENABLE_GPGME=OFF -DENABLE_BOOST=OFF -DENABLE_HARFBUZZ=OFF -DBUILD_GTK_TESTS=OFF -DBUILD_QT5_TESTS=OFF -DBUILD_QT6_TESTS=OFF \
 && cmake --build poppler-build --parallel 2 \
 && cmake --install poppler-build \
 && ldconfig

RUN mkdir -p /runtime/usr/local/bin /runtime/etc /runtime/usr/share /runtime/run/want-keep /runtime/tmp /runtime/usr/share/want-keep/licenses \
 && cp /usr/local/bin/qpdf /usr/local/bin/pdftoppm /runtime/usr/local/bin/ \
 && ldd /usr/local/bin/qpdf /usr/local/bin/pdftoppm > /runtime-dependencies.txt \
 && ! grep -q 'not found' /runtime-dependencies.txt \
 && awk '$2=="=>" && $3 ~ /^\// {print $3} $1 ~ /^\// {print $1}' /runtime-dependencies.txt | sort -u | while read library; do cp --parents -L "$library" /runtime/; done
RUN cp -a /etc/fonts /runtime/etc/ \
 && cp -a /usr/share/fonts /usr/share/fontconfig /usr/share/doc /runtime/usr/share/ \
 && cp /build/qpdf-12.4.1/LICENSE.txt /build/qpdf-12.4.1/NOTICE.md /build/qpdf-12.4.1/Artistic-2.0 /runtime/usr/share/want-keep/licenses/ \
 && cp /build/poppler-26.09.0/COPYING /runtime/usr/share/want-keep/licenses/Poppler-COPYING \
 && dpkg-query -W -f='${Package} ${Version}\n' > /runtime/usr/share/want-keep/licenses/debian-packages.txt \
 && cp /runtime-dependencies.txt /runtime/usr/share/want-keep/licenses/ \
 && chmod 1777 /runtime/tmp \
 && chown 10000:10000 /runtime/run/want-keep \
 && ldconfig -r /runtime /usr/local/lib \
 && chroot /runtime /usr/local/bin/qpdf --version \
 && chroot /runtime /usr/local/bin/pdftoppm -v

FROM tools AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/document-processor ./cmd/document-processor
RUN CGO_ENABLED=0 go build -trimpath -o /out/privacy-faultprobe ./test/integration/privacy/faultprobe

FROM scratch AS runtime
COPY --from=tools /runtime/ /
COPY --from=build /out/document-processor /usr/local/bin/document-processor
USER 10000:10000
ENTRYPOINT ["/usr/local/bin/document-processor"]

FROM runtime AS runtime-faults
COPY --from=build /out/privacy-faultprobe /usr/local/bin/privacy-faultprobe

FROM build AS privacy-test
RUN mkdir -p /run/want-keep && chown 10000:10000 /run/want-keep \
 && go test -race -c -tags=integration -o /out/privacy.test ./test/integration/privacy
USER 10000:10000
ENTRYPOINT ["/out/privacy.test"]
