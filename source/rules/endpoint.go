// Copyright 2024 The Perses Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rules

import (
	"net/http"

	"github.com/labstack/echo/v4"
	persesEcho "github.com/perses/common/echo"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/pkg/analyze/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
)

func NewAPI(db database.Database) persesEcho.Register {
	return &endpoint{
		db: db,
	}
}

type request struct {
	Source string         `json:"source"`
	Groups []v1.RuleGroup `json:"groups"`
}

type endpoint struct {
	db database.Database
}

func (e *endpoint) RegisterRoute(ech *echo.Echo) {
	path := "/api/v1/rules"
	ech.POST(path, e.PushRules)
}

func (e *endpoint) PushRules(ctx echo.Context) error {
	var data request
	if err := ctx.Bind(&data); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	metricUsage, invalidMetricUsage, errs := prometheus.Analyze(data.Groups, data.Source)
	for _, logErr := range errs {
		logErr.Log(logrus.StandardLogger().WithField("endpoint", "rules"))
	}
	if len(metricUsage) > 0 {
		e.db.EnqueueUsage(metricUsage)
	}
	if len(invalidMetricUsage) > 0 {
		e.db.EnqueueInvalidMetricsUsage(invalidMetricUsage)
	}
	return ctx.JSON(http.StatusAccepted, echo.Map{"message": "OK"})
}
