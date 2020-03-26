// Copyright 2020 The Lokomotive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aks

var terraformConfigTmpl = `locals {
  subscription_id           = "{{ .SubscriptionID }}"
  tenant_id                 = "{{ .TenantID }}"
  application_name          = "{{ .ApplicationName }}"
  location                  = "{{ .Location }}"
  resource_group_name       = "{{ .ResourceGroupName }}"
  kubernetes_version        = "1.16.7"
  cluster_name              = "{{ .ClusterName }}"
  default_node_pool_name    = "{{ (index .WorkerPools 0).Name }}"
  default_node_pool_vm_size = "{{ (index .WorkerPools 0).VMSize }}"
  default_node_pool_count   = {{ (index .WorkerPools 0).Count  }}
}

provider "azurerm" {
  version = "2.2.0"

  # https://github.com/terraform-providers/terraform-provider-azurerm/issues/5893
  features {}
}

provider "azuread" {
  version = "0.8.0"
}

provider "random" {
  version = "2.2.1"
}

provider "local" {
  version = "1.4.0"
}

# TODO: user may want to provide their own resource group.
resource "azurerm_resource_group" "aks" {
  name     = local.resource_group_name
  location = local.location
}

# TODO: user may want to provide their own service principle details.
resource "azuread_application" "aks" {
  name = local.application_name
}

resource "azuread_service_principal" "aks" {
  application_id = azuread_application.aks.application_id
}

resource "random_string" "password" {
  length  = 16
  special = true

  override_special = "/@\" "
}

resource "azuread_application_password" "aks" {
  application_object_id = azuread_application.aks.object_id
  value                 = random_string.password.result
  end_date_relative     = "86000h"
}

resource "azurerm_role_assignment" "aks" {
  scope                = "/subscriptions/${local.subscription_id}"
  role_definition_name = "Contributor"
  principal_id         = azuread_service_principal.aks.id
}

resource "azurerm_kubernetes_cluster" "aks" {
  name                = local.cluster_name
  location            = azurerm_resource_group.aks.location
  resource_group_name = azurerm_resource_group.aks.name
  kubernetes_version  = local.kubernetes_version
  dns_prefix          = local.cluster_name

  default_node_pool {
    name       = local.default_node_pool_name
    vm_size    = local.default_node_pool_vm_size
    node_count = local.default_node_pool_count

    {{- if (index .WorkerPools 0).Labels }}
    node_labels = {
      {{- range $k, $v := (index .WorkerPools 0).Labels }}
      "{{ $k }}" = "{{ $v }}"
      {{- end }}
    }
    {{- end }}

    {{- if (index .WorkerPools 0).Taints }}
    node_taints = [
      {{- range (index .WorkerPools 0).Taints }}
      "{{ . }}",
      {{- end }}
    ]
    {{- end }}
  }

  role_based_access_control {
    enabled = true
  }

  service_principal {
    client_id     = azuread_application.aks.application_id
    client_secret = azuread_application_password.aks.value
  }

  network_profile {
    network_plugin = "kubenet"
    network_policy = "calico"
  }

  {{- if .Tags }}
  tags = {
    {{- range $k, $v := .Tags }}
    "{{ $k }}" = "{{ $v }}"
    {{- end }}
  }
  {{- end }}
}

{{ range $index, $pool := (slice .WorkerPools 1) }}
resource "azurerm_kubernetes_cluster_node_pool" "worker-{{ $pool.Name }}" {
  name                  = "{{ $pool.Name }}"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.aks.id
  vm_size               = "{{ $pool.VMSize }}"
  node_count            = "{{ $pool.Count }}"

  {{- if $pool.Labels }}
  node_labels = {
    {{- range $k, $v := $pool.Labels }}
    "{{ $k }}" = "{{ $v }}"
    {{- end }}
  }
  {{- end }}


  {{- if $pool.Taints }}
  node_taints = [
    {{- range $pool.Taints }}
    "{{ . }}",
    {{- end }}
  ]
  {{- end }}


  {{- if $.Tags }}
  tags = {
    {{- range $k, $v := $.Tags }}
    "{{ $k }}" = "{{ $v }}"
    {{- end }}
  }
  {{- end }}
}
{{- end }}

resource "local_file" "kubeconfig" {
  sensitive_content = azurerm_kubernetes_cluster.aks.kube_config_raw
  filename          = "../cluster-assets/auth/kubeconfig"
}

# Stub output, which indicates, that Terraform run at least once.
# Used when checking, if we should ask user for confirmation, when
# applying changes to the cluster.
output "initialized" {
  value = true
}
`
