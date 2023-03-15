/*
Copyright 2023 The KubeVirt CSI driver Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registry

import (
	"fmt"

	"github.com/docker/distribution/reference"
)

func Must(s string, err error) string {
	if err != nil {
		panic(err)
	}

	return s
}

// RewriteImage will apply the given overwriteRegistry to a given docker
// image reference.
func RewriteImage(image, defaultRegistry, overwriteRegistry, overwriteTag string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("invalid reference %q: %w", image, err)
	}

	domain := reference.Domain(named)
	origDomain := domain
	if origDomain == "" {
		origDomain = defaultRegistry
	}

	if overwriteRegistry != "" {
		domain = overwriteRegistry
	}
	if domain == "" {
		domain = defaultRegistry
	}

	// construct name image name
	image = domain + "/" + reference.Path(named)

	tag := ""
	if tagged, ok := named.(reference.Tagged); ok {
		tag = tagged.Tag()
	}
	if overwriteTag != "" {
		tag = overwriteTag
	}
	image += ":" + tag

	// If the registry (domain) has been changed, remove the
	// digest as it's unlikely that a) the repo digest has
	// been kept when mirroring the image and b) the chance
	// of a local registry being poisoned with bad images is
	// much lower anyhow.
	if origDomain == domain {
		if digested, ok := named.(reference.Digested); ok {
			image += "@" + string(digested.Digest())
		}
	}

	return image, nil
}
