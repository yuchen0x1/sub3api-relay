<template>
  <div class="space-y-6">
            <!-- Admin API Key Settings -->
            <div class="card">
              <div
                class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
              >
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.adminApiKey.title") }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.adminApiKey.description") }}
                </p>
              </div>
              <div class="space-y-4 p-6">
                <!-- Security Warning -->
                <div
                  class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
                >
                  <div class="flex items-start">
                    <Icon
                      name="exclamationTriangle"
                      size="md"
                      class="mt-0.5 flex-shrink-0 text-amber-500"
                    />
                    <p class="ml-3 text-sm text-amber-700 dark:text-amber-300">
                      {{ t("admin.settings.adminApiKey.securityWarning") }}
                    </p>
                  </div>
                </div>

                <!-- Loading State -->
                <div
                  v-if="adminApiKeyLoading"
                  class="flex items-center gap-2 text-gray-500"
                >
                  <div
                    class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                  ></div>
                  {{ t("common.loading") }}
                </div>

                <!-- No Key Configured -->
                <div
                  v-else-if="!adminApiKeyExists"
                  class="flex items-center justify-between"
                >
                  <span class="text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.adminApiKey.notConfigured") }}
                  </span>
                  <button
                    type="button"
                    @click="createAdminApiKey"
                    :disabled="adminApiKeyOperating"
                    class="btn btn-primary btn-sm"
                  >
                    <svg
                      v-if="adminApiKeyOperating"
                      class="mr-1 h-4 w-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    {{
                      adminApiKeyOperating
                        ? t("admin.settings.adminApiKey.creating")
                        : t("admin.settings.adminApiKey.create")
                    }}
                  </button>
                </div>

                <!-- Key Exists -->
                <div v-else class="space-y-4">
                  <div class="flex items-center justify-between">
                    <div>
                      <label
                        class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.adminApiKey.currentKey") }}
                      </label>
                      <code
                        class="rounded bg-gray-100 px-2 py-1 font-mono text-sm text-gray-900 dark:bg-dark-700 dark:text-gray-100"
                      >
                        {{ adminApiKeyMasked }}
                      </code>
                    </div>
                    <div class="flex gap-2">
                      <button
                        type="button"
                        @click="regenerateAdminApiKey"
                        :disabled="adminApiKeyOperating"
                        class="btn btn-secondary btn-sm"
                      >
                        {{
                          adminApiKeyOperating
                            ? t("admin.settings.adminApiKey.regenerating")
                            : t("admin.settings.adminApiKey.regenerate")
                        }}
                      </button>
                      <button
                        type="button"
                        @click="deleteAdminApiKey"
                        :disabled="adminApiKeyOperating"
                        class="btn btn-secondary btn-sm text-red-600 hover:text-red-700 dark:text-red-400"
                      >
                        {{ t("admin.settings.adminApiKey.delete") }}
                      </button>
                    </div>
                  </div>

                  <!-- Newly Generated Key Display -->
                  <div
                    v-if="newAdminApiKey"
                    class="space-y-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20"
                  >
                    <p
                      class="text-sm font-medium text-green-700 dark:text-green-300"
                    >
                      {{ t("admin.settings.adminApiKey.keyWarning") }}
                    </p>
                    <div class="flex items-center gap-2">
                      <code
                        class="flex-1 select-all break-all rounded border border-green-300 bg-white px-3 py-2 font-mono text-sm dark:border-green-700 dark:bg-dark-800"
                      >
                        {{ newAdminApiKey }}
                      </code>
                      <button
                        type="button"
                        @click="copyNewKey"
                        class="btn btn-primary btn-sm flex-shrink-0"
                      >
                        {{ t("admin.settings.adminApiKey.copyKey") }}
                      </button>
                    </div>
                    <p class="text-xs text-green-600 dark:text-green-400">
                      {{ t("admin.settings.adminApiKey.usage") }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <!-- Registration Settings -->
            <div class="card">
              <div
                class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
              >
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.registration.title") }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.registration.description") }}
                </p>
              </div>
              <div class="space-y-5 p-6">
                <!-- Enable Registration -->
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.registration.enableRegistration")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{
                        t("admin.settings.registration.enableRegistrationHint")
                      }}
                    </p>
                  </div>
                  <Toggle v-model="form.registration_enabled" />
                </div>

                <!-- Email Verification -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.registration.emailVerification")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.registration.emailVerificationHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.email_verify_enabled" />
                </div>

                <!-- Email Suffix Whitelist -->
                <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.emailSuffixWhitelist")
                  }}</label>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{
                      t("admin.settings.registration.emailSuffixWhitelistHint")
                    }}
                  </p>
                  <div
                    class="mt-3 rounded-lg border border-gray-300 bg-white p-2 dark:border-dark-500 dark:bg-dark-700"
                  >
                    <div class="flex flex-wrap items-center gap-2">
                      <span
                        v-for="suffix in registrationEmailSuffixWhitelistTags"
                        :key="suffix"
                        class="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs font-mono text-gray-700 dark:bg-dark-600 dark:text-gray-200"
                      >
                        <span>{{ suffix }}</span>
                        <button :aria-label="t('common.delete')"
                          type="button"
                          class="rounded-full text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-300 dark:hover:bg-dark-500 dark:hover:text-white"
                          @click="
                            removeRegistrationEmailSuffixWhitelistTag(suffix)
                          "
                        >
                          <Icon
                            name="x"
                            size="xs"
                            class="h-3.5 w-3.5"
                            :stroke-width="2"
                          />
                        </button>
                      </span>

                      <div
                        class="flex min-w-[220px] flex-1 items-center gap-1 rounded border border-transparent px-2 py-1 focus-within:border-primary-300 dark:focus-within:border-primary-700"
                      >
                        <input
                          v-model="registrationEmailSuffixWhitelistDraft"
                          type="text"
                          class="w-full bg-transparent text-sm font-mono text-gray-900 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                          :placeholder="
                            t(
                              'admin.settings.registration.emailSuffixWhitelistPlaceholder',
                            )
                          "
                          @input="
                            handleRegistrationEmailSuffixWhitelistDraftInput
                          "
                          @keydown="
                            handleRegistrationEmailSuffixWhitelistDraftKeydown
                          "
                          @blur="commitRegistrationEmailSuffixWhitelistDraft"
                          @paste="handleRegistrationEmailSuffixWhitelistPaste"
                        />
                      </div>
                    </div>
                  </div>
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        "admin.settings.registration.emailSuffixWhitelistInputHint",
                      )
                    }}
                  </p>
                </div>

                <!-- Promo Code -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.registration.promoCode")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.registration.promoCodeHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.promo_code_enabled" />
                </div>

                <!-- Invitation Code -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.registration.invitationCode")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.registration.invitationCodeHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.invitation_code_enabled" />
                </div>
                <!-- Password Reset - Only show when email verification is enabled -->
                <div
                  v-if="form.email_verify_enabled"
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.registration.passwordReset")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.registration.passwordResetHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.password_reset_enabled" />
                </div>
                <!-- Frontend URL - Only show when password reset is enabled -->
                <div
                  v-if="form.email_verify_enabled && form.password_reset_enabled"
                  class="border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.registration.frontendUrl") }}
                  </label>
                  <input
                    v-model="form.frontend_url"
                    type="url"
                    class="input"
                    :placeholder="
                      t('admin.settings.registration.frontendUrlPlaceholder')
                    "
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.registration.frontendUrlHint") }}
                  </p>
                </div>

                <!-- TOTP 2FA -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.registration.totp")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.registration.totpHint") }}
                    </p>
                    <!-- Warning when encryption key not configured -->
                    <p
                      v-if="!form.totp_encryption_key_configured"
                      class="mt-2 text-sm text-amber-600 dark:text-amber-400"
                    >
                      {{ t("admin.settings.registration.totpKeyNotConfigured") }}
                    </p>
                  </div>
                  <Toggle
                    v-model="form.totp_enabled"
                    :disabled="!form.totp_encryption_key_configured"
                  />
                </div>

                <!-- Passkey sign-in -->
                <div
                  class="border-t border-gray-100 pt-4 dark:border-dark-700"
                  data-testid="passkey-settings"
                >
                  <div class="flex items-start justify-between gap-4">
                    <div>
                      <label class="font-medium text-gray-900 dark:text-white">{{
                        t("admin.settings.security.passkey")
                      }}</label>
                      <p class="text-sm text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.security.passkeyHint") }}
                      </p>
                    </div>
                    <Toggle
                      v-model="form.passkey_enabled"
                      data-testid="passkey-toggle"
                      :disabled="!form.passkey_configured"
                    />
                  </div>
                  <div
                    class="mt-3 rounded-lg border px-3 py-2 text-sm"
                    :class="
                      form.passkey_configured
                        ? 'border-green-200 bg-green-50 text-green-800 dark:border-green-900 dark:bg-green-950/40 dark:text-green-300'
                        : 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300'
                    "
                    data-testid="passkey-config-status"
                  >
                    <p class="font-medium">
                      {{
                        form.passkey_configured
                          ? t("admin.settings.security.passkeyConfigured")
                          : t("admin.settings.security.passkeyNotConfigured")
                      }}
                    </p>
                    <p class="mt-1 break-all">
                      {{ t("admin.settings.security.passkeyRPID") }}:
                      {{
                        form.passkey_rp_id ||
                        t("admin.settings.security.passkeyValueNotConfigured")
                      }}
                    </p>
                    <p class="mt-1 break-all">
                      {{ t("admin.settings.security.passkeyOrigins") }}:
                      {{
                        form.passkey_rp_origins.length > 0
                          ? form.passkey_rp_origins.join(", ")
                          : t(
                              "admin.settings.security.passkeyValueNotConfigured",
                            )
                      }}
                    </p>
                    <p v-if="!form.passkey_configured" class="mt-2">
                      {{ t("admin.settings.security.passkeyDeploymentHint") }}
                    </p>
                  </div>
                </div>

                <!-- 敏感操作 step-up 2FA -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.security.stepUp")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.security.stepUpHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.step_up_enabled" />
                </div>

                <!-- 会话 IP/UA 绑定 -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.security.sessionBinding")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.security.sessionBindingHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.session_binding_enabled" />
                </div>

                <!-- 审计日志保留天数 -->
                <div
                  class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.security.auditRetention")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.security.auditRetentionHint") }}
                    </p>
                  </div>
                  <input
                    v-model.number="form.audit_log_retention_days"
                    type="number"
                    min="0"
                    class="input w-28 text-right"
                  />
                </div>
              </div>
            </div>

            <!-- API Key IP ACL Settings -->
            <div class="card">
              <div
                class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
              >
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.apiKeyAcl.title") }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.apiKeyAcl.description") }}
                </p>
              </div>
              <div class="space-y-5 p-6">
                <div class="flex items-center justify-between gap-4">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t("admin.settings.apiKeyAcl.trustForwardedIp") }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.apiKeyAcl.trustForwardedIpHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.api_key_acl_trust_forwarded_ip" />
                </div>

                <div
                  v-if="form.api_key_acl_trust_forwarded_ip"
                  class="border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <label
                    for="forwarded-client-ip-headers"
                    class="font-medium text-gray-900 dark:text-white"
                  >
                    {{ t("admin.settings.apiKeyAcl.forwardedClientIpHeaders") }}
                  </label>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.apiKeyAcl.forwardedClientIpHeadersHint") }}
                  </p>
                  <div
                    class="mt-3 rounded-lg border border-gray-300 bg-white p-2 dark:border-dark-500 dark:bg-dark-700"
                  >
                    <div class="flex flex-wrap items-center gap-2">
                      <span
                        v-for="header in form.forwarded_client_ip_headers"
                        :key="header"
                        data-testid="forwarded-client-ip-header-tag"
                        class="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs font-mono text-gray-700 dark:bg-dark-600 dark:text-gray-200"
                      >
                        <span>{{ header }}</span>
                        <button
                          type="button"
                          class="rounded-full text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-300 dark:hover:bg-dark-500 dark:hover:text-white"
                          :aria-label="t('admin.settings.apiKeyAcl.removeForwardedClientIpHeader', { header })"
                          @click="removeForwardedClientIpHeader(header)"
                        >
                          <Icon
                            name="x"
                            size="xs"
                            class="h-3.5 w-3.5"
                            :stroke-width="2"
                          />
                        </button>
                      </span>
                      <div
                        class="flex min-w-[220px] flex-1 items-center gap-1 rounded border border-transparent px-2 py-1 focus-within:border-primary-300 dark:focus-within:border-primary-700"
                      >
                        <input
                          id="forwarded-client-ip-headers"
                          v-model="forwardedClientIpHeaderDraft"
                          data-testid="forwarded-client-ip-headers-input"
                          type="text"
                          class="w-full bg-transparent text-sm font-mono text-gray-900 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                          :placeholder="t('admin.settings.apiKeyAcl.forwardedClientIpHeadersPlaceholder')"
                          @keydown="handleForwardedClientIpHeaderKeydown"
                          @blur="commitForwardedClientIpHeaderDraft"
                          @paste="handleForwardedClientIpHeaderPaste"
                        />
                      </div>
                    </div>
                  </div>
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.apiKeyAcl.forwardedClientIpHeadersRiskHint") }}
                  </p>
                </div>
              </div>
            </div>

            <!-- Panel API Rate Limit Settings -->
            <div class="card">
              <div
                class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
              >
                <div class="flex items-center gap-2">
                  <Icon
                    name="shield"
                    size="md"
                    class="text-primary-500"
                  />
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                    {{ t("admin.settings.panelRateLimit.title") }}
                  </h2>
                </div>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.panelRateLimit.description") }}
                </p>
              </div>
              <div class="space-y-5 p-6">
                <div
                  v-if="panelRateLimitLoading"
                  class="flex items-center gap-2 text-gray-500"
                >
                  <div
                    class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                  ></div>
                  {{ t("common.loading") }}
                </div>

                <template v-else>
                  <!-- 计数维度说明：按账号计数，反代部署无误伤 -->
                  <div
                    class="rounded-lg border border-sky-200 bg-sky-50 p-4 dark:border-sky-800 dark:bg-sky-900/20"
                  >
                    <div class="flex items-start">
                      <Icon
                        name="infoCircle"
                        size="md"
                        class="mt-0.5 flex-shrink-0 text-sky-500"
                      />
                      <p class="ml-3 text-sm text-sky-700 dark:text-sky-300">
                        {{ t("admin.settings.panelRateLimit.proxySafeNote") }}
                      </p>
                    </div>
                  </div>

                  <div class="flex items-center justify-between">
                    <div>
                      <label class="font-medium text-gray-900 dark:text-white">{{
                        t("admin.settings.panelRateLimit.enabled")
                      }}</label>
                      <p class="text-sm text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.panelRateLimit.enabledHint") }}
                      </p>
                    </div>
                    <Toggle v-model="panelRateLimitForm.enabled" />
                  </div>

                  <div
                    v-if="panelRateLimitForm.enabled"
                    class="space-y-5 border-t border-gray-100 pt-4 dark:border-dark-700"
                  >
                    <div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
                      <div>
                        <label
                          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                        >
                          {{ t("admin.settings.panelRateLimit.userRpm") }}
                        </label>
                        <div class="flex items-center gap-2">
                          <input
                            v-model.number="panelRateLimitForm.user_rpm"
                            data-testid="panel-rate-limit-user-rpm"
                            type="number"
                            min="0"
                            max="100000"
                            class="input w-32"
                          />
                          <span class="text-sm text-gray-500 dark:text-gray-400">
                            {{ t("admin.settings.panelRateLimit.perMinute") }}
                          </span>
                        </div>
                        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.userRpmHint") }}
                        </p>
                      </div>

                      <div>
                        <label
                          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                        >
                          {{ t("admin.settings.panelRateLimit.heavyRpm") }}
                        </label>
                        <div class="flex items-center gap-2">
                          <input
                            v-model.number="panelRateLimitForm.heavy_rpm"
                            type="number"
                            min="0"
                            max="100000"
                            class="input w-32"
                          />
                          <span class="text-sm text-gray-500 dark:text-gray-400">
                            {{ t("admin.settings.panelRateLimit.perMinute") }}
                          </span>
                        </div>
                        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.heavyRpmHint") }}
                        </p>
                      </div>

                      <div>
                        <label
                          class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                        >
                          {{ t("admin.settings.panelRateLimit.publicIpRpm") }}
                        </label>
                        <div class="flex items-center gap-2">
                          <input
                            v-model.number="panelRateLimitForm.public_ip_rpm"
                            type="number"
                            min="0"
                            max="100000"
                            class="input w-32"
                          />
                          <span class="text-sm text-gray-500 dark:text-gray-400">
                            {{ t("admin.settings.panelRateLimit.perMinute") }}
                          </span>
                        </div>
                        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.publicIpRpmHint") }}
                        </p>
                      </div>
                    </div>

                    <div
                      class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                    >
                      <div>
                        <label class="font-medium text-gray-900 dark:text-white">{{
                          t("admin.settings.panelRateLimit.exemptAdmin")
                        }}</label>
                        <p class="text-sm text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.exemptAdminHint") }}
                        </p>
                      </div>
                      <Toggle v-model="panelRateLimitForm.exempt_admin" />
                    </div>
                  </div>

                  <div
                    class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                  >
                    <button
                      type="button"
                      data-testid="panel-rate-limit-save"
                      @click="savePanelRateLimitSettings"
                      :disabled="panelRateLimitSaving"
                      class="btn btn-primary btn-sm"
                    >
                      <svg
                        v-if="panelRateLimitSaving"
                        class="mr-1 h-4 w-4 animate-spin"
                        fill="none"
                        viewBox="0 0 24 24"
                      >
                        <circle
                          class="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          stroke-width="4"
                        ></circle>
                        <path
                          class="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                        ></path>
                      </svg>
                      {{
                        panelRateLimitSaving
                          ? t("common.saving")
                          : t("common.save")
                      }}
                    </button>
                  </div>
                </template>
              </div>
            </div>

            <!-- Cloudflare Turnstile Settings -->
            <div class="card">
              <div
                class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
              >
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.turnstile.title") }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.turnstile.description") }}
                </p>
              </div>
              <div class="space-y-5 p-6">
                <!-- Enable Turnstile -->
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.turnstile.enableTurnstile")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.turnstile.enableTurnstileHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.turnstile_enabled" />
                </div>

                <!-- Turnstile Keys - Only show when enabled -->
                <div
                  v-if="form.turnstile_enabled"
                  class="border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div class="grid grid-cols-1 gap-6">
                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.turnstile.siteKey") }}
                      </label>
                      <input
                        v-model="form.turnstile_site_key"
                        type="text"
                        class="input font-mono text-sm"
                        placeholder="0x4AAAAAAA..."
                      />
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.turnstile.siteKeyHint") }}
                        <a
                          href="https://dash.cloudflare.com/"
                          target="_blank"
                          class="text-primary-600 hover:text-primary-500"
                          >{{
                            t("admin.settings.turnstile.cloudflareDashboard")
                          }}</a
                        >
                      </p>
                    </div>
                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.turnstile.secretKeyLabel") }}
                      </label>
                      <input
                        v-model="form.turnstile_secret_key"
                        type="password"
                        class="input font-mono text-sm"
                        placeholder="0x4AAAAAAA..."
                      />
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{
                          form.turnstile_secret_key_configured
                            ? t(
                                "admin.settings.turnstile.secretKeyConfiguredHint",
                              )
                            : t("admin.settings.turnstile.secretKeyHint")
                        }}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>

  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api";
import Icon from "@/components/icons/Icon.vue";
import Toggle from "@/components/common/Toggle.vue";
import type { SettingsForm } from "./settingsForm";
import { extractApiErrorMessage } from "@/utils/apiError";
import { useAppStore } from "@/stores";
import {
  isRegistrationEmailSuffixDomainValid,
  normalizeRegistrationEmailSuffixDomain,
  parseRegistrationEmailSuffixWhitelistInput,
} from "@/utils/registrationEmailPolicy";

const props = defineProps<{
  form: SettingsForm;
}>();
// Vue 3.4：props 解构不具响应性。form 是父层稳定的 reactive 对象引用，
// 解构后就地修改嵌套字段仍会写回父层表单。
const { form } = props;

// 注册邮箱后缀白名单由父组件持有（保存时用于构造 payload），
// 子组件通过 v-model 双向同步，父级 load/save 后整体替换即自动回填。
const registrationEmailSuffixWhitelistTags = defineModel<string[]>(
  "emailSuffixWhitelist",
  { required: true },
);

const { t } = useI18n();
const appStore = useAppStore();



// ==================== 注册邮箱后缀白名单 ====================

const registrationEmailSuffixWhitelistDraft = ref("");
const registrationEmailSuffixWhitelistSeparatorKeys = new Set([
  " ",
  ",",
  "，",
  "Enter",
  "Tab",
]);

watch(registrationEmailSuffixWhitelistTags, () => {
  // 父组件在 load/save 后整体替换数组时，清空本地输入草稿
  registrationEmailSuffixWhitelistDraft.value = "";
});

function removeRegistrationEmailSuffixWhitelistTag(suffix: string) {
  registrationEmailSuffixWhitelistTags.value =
    registrationEmailSuffixWhitelistTags.value.filter(
      (item) => item !== suffix,
    );
}

function addRegistrationEmailSuffixWhitelistTag(raw: string) {
  const suffix = normalizeRegistrationEmailSuffixDomain(raw);
  if (
    !isRegistrationEmailSuffixDomainValid(suffix) ||
    registrationEmailSuffixWhitelistTags.value.includes(suffix)
  ) {
    return;
  }
  registrationEmailSuffixWhitelistTags.value = [
    ...registrationEmailSuffixWhitelistTags.value,
    suffix,
  ];
}

function commitRegistrationEmailSuffixWhitelistDraft() {
  if (!registrationEmailSuffixWhitelistDraft.value) {
    return;
  }
  addRegistrationEmailSuffixWhitelistTag(
    registrationEmailSuffixWhitelistDraft.value,
  );
  registrationEmailSuffixWhitelistDraft.value = "";
}

function handleRegistrationEmailSuffixWhitelistDraftInput() {
  registrationEmailSuffixWhitelistDraft.value =
    normalizeRegistrationEmailSuffixDomain(
      registrationEmailSuffixWhitelistDraft.value,
    );
}

function handleRegistrationEmailSuffixWhitelistDraftKeydown(
  event: KeyboardEvent,
) {
  if (event.isComposing) {
    return;
  }

  if (registrationEmailSuffixWhitelistSeparatorKeys.has(event.key)) {
    event.preventDefault();
    commitRegistrationEmailSuffixWhitelistDraft();
    return;
  }

  if (
    event.key === "Backspace" &&
    !registrationEmailSuffixWhitelistDraft.value &&
    registrationEmailSuffixWhitelistTags.value.length > 0
  ) {
    registrationEmailSuffixWhitelistTags.value.pop();
  }
}

function handleRegistrationEmailSuffixWhitelistPaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData("text") || "";
  if (!text.trim()) {
    return;
  }
  event.preventDefault();
  const tokens = parseRegistrationEmailSuffixWhitelistInput(text);
  for (const token of tokens) {
    addRegistrationEmailSuffixWhitelistTag(token);
  }
}

// ==================== API Key ACL：可信转发 IP 头 ====================

const forwardedClientIpHeaderDraft = ref("");
const forwardedClientIpHeaderSeparatorKeys = new Set([
  " ",
  ",",
  "，",
  "Enter",
  "Tab",
]);
const forwardedClientIpHeaderTokenPattern = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;
const maxForwardedClientIpHeaders = 16;

type ForwardedClientIpHeaderResult = "added" | "duplicate" | "invalid" | "full";

watch(
  () => form.forwarded_client_ip_headers,
  () => {
    // 父组件在 load/save 后整体替换数组时，清空本地输入草稿
    forwardedClientIpHeaderDraft.value = "";
  },
);

function normalizeForwardedClientIpHeader(raw: string): string {
  const header = raw.trim();
  if (!forwardedClientIpHeaderTokenPattern.test(header)) {
    return "";
  }

  return header
    .toLowerCase()
    .split("-")
    .map((part) => `${part.charAt(0).toUpperCase()}${part.slice(1)}`)
    .join("-");
}


function removeForwardedClientIpHeader(header: string) {
  form.forwarded_client_ip_headers = form.forwarded_client_ip_headers.filter(
    (item) => item !== header,
  );
}

function addForwardedClientIpHeader(raw: string): ForwardedClientIpHeaderResult {
  const header = normalizeForwardedClientIpHeader(raw);
  if (!header) {
    return "invalid";
  }
  if (
    form.forwarded_client_ip_headers.some(
      (item) => item.toLowerCase() === header.toLowerCase(),
    )
  ) {
    return "duplicate";
  }
  if (form.forwarded_client_ip_headers.length >= maxForwardedClientIpHeaders) {
    return "full";
  }
  form.forwarded_client_ip_headers = [
    ...form.forwarded_client_ip_headers,
    header,
  ];
  return "added";
}

function showForwardedClientIpHeaderError(result: ForwardedClientIpHeaderResult) {
  if (result === "invalid") {
    appStore.showError(t("admin.settings.apiKeyAcl.forwardedClientIpHeaderInvalid"));
  } else if (result === "full") {
    appStore.showError(
      t("admin.settings.apiKeyAcl.forwardedClientIpHeadersLimit", {
        max: maxForwardedClientIpHeaders,
      }),
    );
  }
}

function commitForwardedClientIpHeaderDraft() {
  const draft = forwardedClientIpHeaderDraft.value;
  if (!draft) {
    return;
  }
  const result = addForwardedClientIpHeader(draft);
  showForwardedClientIpHeaderError(result);
  forwardedClientIpHeaderDraft.value = "";
}

function handleForwardedClientIpHeaderKeydown(event: KeyboardEvent) {
  if (event.isComposing) {
    return;
  }
  if (forwardedClientIpHeaderSeparatorKeys.has(event.key)) {
    event.preventDefault();
    commitForwardedClientIpHeaderDraft();
    return;
  }
  if (
    event.key === "Backspace" &&
    !forwardedClientIpHeaderDraft.value &&
    form.forwarded_client_ip_headers.length > 0
  ) {
    form.forwarded_client_ip_headers.pop();
  }
}

function handleForwardedClientIpHeaderPaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData("text") || "";
  if (!text.trim()) {
    return;
  }
  event.preventDefault();

  let error: ForwardedClientIpHeaderResult | undefined;
  for (const token of text.split(/[,，;\r\n]+/)) {
    if (!token.trim()) {
      continue;
    }
    const result = addForwardedClientIpHeader(token);
    if (result === "invalid" || result === "full") {
      error = result;
    }
  }
  if (error) {
    showForwardedClientIpHeaderError(error);
  }
}

// ==================== Admin API Key ====================

const adminApiKeyLoading = ref(true);
const adminApiKeyExists = ref(false);
const adminApiKeyMasked = ref("");
const adminApiKeyOperating = ref(false);
const newAdminApiKey = ref("");

async function loadAdminApiKey() {
  adminApiKeyLoading.value = true;
  try {
    const status = await adminAPI.settings.getAdminApiKey();
    adminApiKeyExists.value = status.exists;
    adminApiKeyMasked.value = status.masked_key;
  } catch (_error: unknown) {
    // Silent fail - admin API key status is non-critical
  } finally {
    adminApiKeyLoading.value = false;
  }
}

async function createAdminApiKey() {
  adminApiKeyOperating.value = true;
  try {
    const result = await adminAPI.settings.regenerateAdminApiKey();
    newAdminApiKey.value = result.key;
    adminApiKeyExists.value = true;
    adminApiKeyMasked.value =
      result.key.substring(0, 10) + "..." + result.key.slice(-4);
    appStore.showSuccess(t("admin.settings.adminApiKey.keyGenerated"));
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t("common.error")));
  } finally {
    adminApiKeyOperating.value = false;
  }
}

async function regenerateAdminApiKey() {
  if (!confirm(t("admin.settings.adminApiKey.regenerateConfirm"))) return;
  await createAdminApiKey();
}

async function deleteAdminApiKey() {
  if (!confirm(t("admin.settings.adminApiKey.deleteConfirm"))) return;
  adminApiKeyOperating.value = true;
  try {
    await adminAPI.settings.deleteAdminApiKey();
    adminApiKeyExists.value = false;
    adminApiKeyMasked.value = "";
    newAdminApiKey.value = "";
    appStore.showSuccess(t("admin.settings.adminApiKey.keyDeleted"));
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t("common.error")));
  } finally {
    adminApiKeyOperating.value = false;
  }
}

function copyNewKey() {
  navigator.clipboard
    .writeText(newAdminApiKey.value)
    .then(() => {
      appStore.showSuccess(t("admin.settings.adminApiKey.keyCopied"));
    })
    .catch(() => {
      appStore.showError(t("common.copyFailed"));
    });
}

// ==================== Panel API Rate Limit ====================

const panelRateLimitLoading = ref(true);
const panelRateLimitSaving = ref(false);
const panelRateLimitForm = reactive({
  enabled: true,
  user_rpm: 240,
  heavy_rpm: 60,
  exempt_admin: true,
  public_ip_rpm: 300,
});

async function loadPanelRateLimitSettings() {
  panelRateLimitLoading.value = true;
  try {
    const settings = await adminAPI.settings.getPanelRateLimitSettings();
    Object.assign(panelRateLimitForm, settings);
  } catch (_error: unknown) {
    // Silent fail - settings will use defaults
  } finally {
    panelRateLimitLoading.value = false;
  }
}

async function savePanelRateLimitSettings() {
  panelRateLimitSaving.value = true;
  try {
    const updated = await adminAPI.settings.updatePanelRateLimitSettings({
      enabled: panelRateLimitForm.enabled,
      user_rpm: panelRateLimitForm.user_rpm,
      heavy_rpm: panelRateLimitForm.heavy_rpm,
      exempt_admin: panelRateLimitForm.exempt_admin,
      public_ip_rpm: panelRateLimitForm.public_ip_rpm,
    });
    Object.assign(panelRateLimitForm, updated);
    appStore.showSuccess(t("admin.settings.panelRateLimit.saved"));
  } catch (error: unknown) {
    appStore.showError(
      extractApiErrorMessage(
        error,
        t("admin.settings.panelRateLimit.saveFailed"),
      ),
    );
  } finally {
    panelRateLimitSaving.value = false;
  }
}


onMounted(() => {
  loadAdminApiKey();
  loadPanelRateLimitSettings();
});
</script>

