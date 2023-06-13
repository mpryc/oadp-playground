#
# Copyright Red Hat
#
#    Licensed under the Apache License, Version 2.0 (the "License"); you may
#    not use this file except in compliance with the License. You may obtain
#    a copy of the License at
#
#         http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
#    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
#    License for the specific language governing permissions and limitations
#    under the License.
#

# Variable setup and preflight checks

# may override with environment variable
CONTAINER_ENGINE?=podman
GO_CONTAINER_IMAGE?=docker.io/golang:1.20
PYTHON_BINARY?=python3

ifndef OADP_VENV
  OADP_VENV=.venv
endif

SYS_PYTHON_VER=$(shell $(PYTHON_BINARY) -c 'from sys import version_info; \
  print("%d.%d" % version_info[0:2])')
$(info Found system python version: $(SYS_PYTHON_VER));
PYTHON_VER_CHECK=$(shell $(PYTHON_BINARY) scripts/python-version-check.py)

ifneq ($(strip $(PYTHON_VER_CHECK)),)
  $(error $(PYTHON_VER_CHECK). You may set the PYTHON_BINARY env var to specify a compatible version)
endif

SHELLCHECK=$(shell which shellcheck)

.PHONY: default
default: \
  dev-env

.PHONY: all
all: default

# note the following is required for the makefile help
## TARGET: DESCRIPTION
## ------: -----------
## help: print each make target with a description
.PHONY: help
help:
	@echo ""
	@(printf ""; sed -n 's/^## //p' Makefile) | column -t -s :


# Environment setup

.PHONY: go-container-image
go-container-image:
	${CONTAINER_ENGINE} pull ${GO_CONTAINER_IMAGE}

$(OADP_VENV):go-container-image
	test -d ${OADP_VENV} || ${PYTHON_BINARY} -m venv ${OADP_VENV}
	. ${OADP_VENV}/bin/activate && \
	       pip install -U pip && \
	       pip install -r ./requirements-dev.txt
	touch ${OADP_VENV}

## cli_dev_tools: install all necessary CLI dev tools
.PHONY: cli_dev_tools
cli_dev_tools: $(OADP_VENV)
	. ${OADP_VENV}/bin/activate && \
		./scripts/install_dev_tools.sh -v $(OADP_VENV)

## dev-env: set up everything needed for development (install tools, set up virtual environment, git configuration)
dev-env: $(OADP_VENV) cli_dev_tools
	@echo
	@echo "**** To run VENV:"
	@echo "      $$ source ${OADP_VENV}/bin/activate"
	@echo
	@echo "**** To later deactivate VENV:"
	@echo "      $$ deactivate"
	@echo

## shellcheck: run various tests on a shell scripts
shellcheck: $(OADP_VENV) $(OADP_VENV)/bin/shellcheck
	. ${OADP_VENV}/bin/activate && \
	if [[ -z shellcheck ]]; then echo "Shellcheck is not installed" >&2; false; fi && \
	echo "🐚 📋 Linting shell scripts with shellcheck" && \
	shellcheck $(shell find . -name '*.sh' -type f | grep -v 'venv/\|git/\|.pytest_cache/\|htmlcov/\|_test/test_helper/\|_test/bats\|_test/conftest')


# Cleanup

## clean-dev-env: remove the virtual environment, clean up all .pyc files and container image(s)
clean-dev-env:
	rm -rf ${OADP_VENV}
	find . -iname "*.pyc" -delete
	${CONTAINER_ENGINE} rmi ${GO_CONTAINER_IMAGE}
