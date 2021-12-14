.PHONY: skopeo.build
skopeo.build:  ## Builds the skopeo static binary
skopeo.build: skopeo/static/skopeo-$(GOOS)-$(GOARCH)$(if $(filter $(GOOS),windows),.exe)

.PHONY: skopeo.build.all
skopeo.build.all:
	$(MAKE) --no-print-directory GOOS=linux GOARCH=amd64 skopeo.build
	$(MAKE) --no-print-directory GOOS=linux GOARCH=arm64 skopeo.build
	$(MAKE) --no-print-directory GOOS=darwin GOARCH=amd64 skopeo.build
	$(MAKE) --no-print-directory GOOS=darwin GOARCH=arm64 skopeo.build
	$(MAKE) --no-print-directory GOOS=windows GOARCH=amd64 skopeo.build
	$(MAKE) --no-print-directory GOOS=windows GOARCH=arm64 skopeo.build

skopeo/static/skopeo-$(GOOS)-$(GOARCH)$(if $(filter $(GOOS),windows),.exe): ; $(info $(M) Building skopeo for $(GOOS)/$(GOARCH))
	mkdir -p $(dir $@)
	cd skopeo-static && \
		CGO_ENABLED=0 go build -o $(REPO_ROOT)/$@ -tags containers_image_openpgp github.com/containers/skopeo/cmd/skopeo
