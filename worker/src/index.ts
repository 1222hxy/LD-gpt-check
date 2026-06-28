export interface Env {
  DB: D1Database;
  BASE_URL: string;
  LINUXDO_CLIENT_ID: string;
  LINUXDO_CLIENT_SECRET: string;
  LINUXDO_AUTH_URL: string;
  LINUXDO_TOKEN_URL: string;
  LINUXDO_USERINFO_URL: string;
  TOKEN_SECRET?: string;
  SESSION_SECRET?: string;
  ALLOWED_ORIGINS?: string;
  TURNSTILE_SITE_KEY?: string;
  TURNSTILE_SECRET_KEY?: string;
  ADMIN_LINUXDO_IDS?: string;
  ADMIN_USER_IDS?: string;
  BRIDGE_AI_BASE_URL?: string;
  BRIDGE_AI_API_KEY?: string;
  BRIDGE_AI_MODEL?: string;
}

const DEVICE_EXPIRES_SECONDS = 600;
const POLL_INTERVAL_SECONDS = 3;
const ACCESS_TOKEN_EXPIRES_SECONDS = 180 * 86400;
const MAX_QUESTIONS = 50;
const MAX_ATTEMPTS = 500;
const MAX_STRING_LENGTH = 128;
const MAX_BASE_URL_LENGTH = 256;
const MAX_URL_LENGTH = 512;
const MAX_PREVIEW_LENGTH = 300;
const MAX_METADATA_LENGTH = 2048;
const MAX_JSON_BYTES = 256 * 1024;
const DEFAULT_ADMIN_LINUXDO_IDS = "29368";
const DEFAULT_QUESTION_BANK_SLUG = "default";

const DEFAULT_QUESTION_BANK = {
  schema_version: "1",
  questions: [
    {
      id: "candy_21",
      version: "1",
      title: "糖果形状口味保证题",
      prompt:
        "不使用任何外部工具回答以下问题：\n\n在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）\n\n          苹果味 桃子味 西瓜味\n圆形        7      9      8\n五角星形    7      6      4",
      tags: ["math", "pigeonhole"],
      grader: {
        type: "number",
        expected: "21",
        independent_match: true,
      },
    },
  ],
};

class APIError extends Error {
  constructor(
    public status: number,
    public code: string,
    public publicMessage: string
  ) {
    super(publicMessage);
  }
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const requestID = request.headers.get("cf-ray") || crypto.randomUUID();
    try {
      if (request.method === "OPTIONS") return withCommonHeaders(new Response(null, { status: 204 }), request, env, requestID);

      const url = new URL(request.url);
      const path = url.pathname;

      if (request.method === "GET" && matches(path, "/", "/account")) return withCommonHeaders(await accountPage(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/admin") return withCommonHeaders(await adminPage(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/health") return withCommonHeaders(json({ ok: true }), request, env, requestID);
      if (request.method === "GET" && matches(path, "/api/questions", "/api/v1/questions")) return withCommonHeaders(await publicQuestions(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/api/v1/admin/questions") return withCommonHeaders(await adminQuestionsGet(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/api/v1/admin/questions") return withCommonHeaders(await adminQuestionsPost(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/api/v1/admin/bridges") return withCommonHeaders(await adminBridgesGet(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/api/v1/admin/bridges") return withCommonHeaders(await adminBridgesPost(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/api/v1/admin/bridges/identify") return withCommonHeaders(await adminBridgeIdentifyPost(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/api/v1/bridge-suggestions") return withCommonHeaders(await bridgeSuggestionPost(request, env), request, env, requestID);
      if (request.method === "POST" && matches(path, "/api/device/start", "/api/v1/device-authorizations")) return withCommonHeaders(await deviceStart(request, env), request, env, requestID);
      if (request.method === "POST" && matches(path, "/api/device/poll", "/api/v1/device-authorizations/token")) return withCommonHeaders(await devicePoll(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/device") return withCommonHeaders(await devicePage(request, env), request, env, requestID);
      if (request.method === "POST" && matches(path, "/api/device/approve", "/api/v1/device-authorizations/approve")) return withCommonHeaders(await deviceApprove(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/auth/linuxdo/start") return withCommonHeaders(await oauthStart(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/auth/linuxdo/callback") return withCommonHeaders(await oauthCallback(request, env), request, env, requestID);
      if (request.method === "GET" && matches(path, "/api/me", "/api/v1/me")) return withCommonHeaders(await apiMe(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/api/v1/submissions") return withCommonHeaders(await createSubmission(request, env), request, env, requestID);
      if (request.method === "GET" && path === "/api/v1/submissions") return withCommonHeaders(await listSubmissions(request, env), request, env, requestID);
      if (request.method === "DELETE" && path === "/api/v1/submissions") return withCommonHeaders(await deleteAllSubmissions(request, env), request, env, requestID);
      if (request.method === "DELETE" && submissionItemPath(path)) return withCommonHeaders(await deleteSubmission(request, env), request, env, requestID);
      if (matches(path, "/api/runs", "/api/v1/runs")) return withCommonHeaders(jsonError("runs API is gone; use /api/v1/submissions", 410, "gone", requestID), request, env, requestID);
      if (request.method === "POST" && matches(path, "/api/logout", "/api/v1/sessions/logout")) return withCommonHeaders(await apiLogout(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/logout") return withCommonHeaders(await webLogout(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/account/submissions/delete") return withCommonHeaders(await webDeleteSubmission(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/account/submissions/delete-all") return withCommonHeaders(await webDeleteAllSubmissions(request, env), request, env, requestID);
      if (request.method === "POST" && path === "/account/bridge-suggestions") return withCommonHeaders(await webBridgeSuggestionPost(request, env), request, env, requestID);

      if (knownPath(path)) return withCommonHeaders(jsonError("method not allowed", 405, "method_not_allowed", requestID), request, env, requestID);
      return withCommonHeaders(jsonError("not found", 404, "not_found", requestID), request, env, requestID);
    } catch (err) {
      if (err instanceof APIError) {
        return withCommonHeaders(jsonError(err.publicMessage, err.status, err.code, requestID), request, env, requestID);
      }
      console.error("request failed", { requestID, error: err instanceof Error ? err.message : String(err) });
      return withCommonHeaders(jsonError("internal error", 500, "internal_error", requestID), request, env, requestID);
    }
  },
};

async function publicQuestions(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const requestedSlug = str(url.searchParams.get("slug") || "", MAX_STRING_LENGTH).trim();
  const row = requestedSlug
    ? await env.DB.prepare(
        `SELECT slug, title, schema_version, questions_json, updated_at
         FROM question_banks
         WHERE slug = ? AND is_active = 1`
      )
        .bind(requestedSlug)
        .first<any>()
    : await env.DB.prepare(
        `SELECT slug, title, schema_version, questions_json, updated_at
         FROM question_banks
         WHERE is_active = 1
         ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END, updated_at DESC
         LIMIT 1`
      )
        .bind(DEFAULT_QUESTION_BANK_SLUG)
        .first<any>();
  if (!row && requestedSlug) {
    return jsonError("question bank not found", 404, "not_found");
  }
  if (!row) {
    return json(DEFAULT_QUESTION_BANK);
  }
  try {
    const parsed = JSON.parse(row.questions_json);
    const validation = validateQuestionBank(parsed);
    if (validation) throw new Error(validation);
    return json(parsed);
  } catch (err) {
    console.error("stored question bank is invalid; falling back to default", {
      slug: row.slug,
      error: err instanceof Error ? err.message : String(err),
    });
    return json(DEFAULT_QUESTION_BANK);
  }
}

async function adminQuestionsGet(request: Request, env: Env): Promise<Response> {
  const user = await getWebUser(request, env);
  if (!user) {
    return json({ error: "login required", code: "unauthorized", login_url: `/auth/linuxdo/start?next=${encodeURIComponent("/admin/questions")}` }, 401);
  }
  if (!isAdminUser(user, env)) return jsonError("forbidden", 403, "forbidden");
  const row = await loadEditableQuestionBank(env);
  return json({
    user: publicUser(user),
    bank: {
      slug: row?.slug || DEFAULT_QUESTION_BANK_SLUG,
      title: row?.title || "Default question bank",
      schema_version: row?.schema_version || "1",
      questions_json: row?.questions_json || JSON.stringify(DEFAULT_QUESTION_BANK, null, 2),
      is_active: row?.is_active == null ? true : !!row.is_active,
      updated_at: row?.updated_at || "",
    },
    public_url: "/api/v1/questions",
  });
}

async function adminQuestionsPost(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return jsonError("login required", 401, "unauthorized");
  if (!isAdminUser(user, env)) return jsonError("forbidden", 403, "forbidden");

  const body = await readJson<any>(request);
  const slug = str(body.slug || DEFAULT_QUESTION_BANK_SLUG, 64).trim() || DEFAULT_QUESTION_BANK_SLUG;
  if (!/^[a-zA-Z0-9_.-]{1,64}$/.test(slug)) return jsonError("slug is invalid", 400, "bad_request");
  const title = str(body.title || "Default question bank", MAX_STRING_LENGTH).trim() || "Default question bank";
  const isActive = body.is_active === false ? 0 : 1;
  const jsonText = typeof body.questions_json === "string" ? body.questions_json : JSON.stringify(body.questions_json ?? "");
  if (new TextEncoder().encode(jsonText).length > MAX_JSON_BYTES) return jsonError("question bank is too large", 413, "payload_too_large");
  let parsed: any;
  try {
    parsed = JSON.parse(jsonText);
  } catch {
    return jsonError("questions_json is invalid JSON", 400, "bad_request");
  }
  const validation = validateQuestionBank(parsed);
  if (validation) return jsonError(validation, 422, "validation_failed");

  const normalizedJSON = JSON.stringify(parsed, null, 2);
  const now = iso(new Date());
  await env.DB.prepare(
    `INSERT INTO question_banks
       (id, slug, title, schema_version, questions_json, is_active, created_by, updated_by, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
     ON CONFLICT(slug) DO UPDATE SET
       title = excluded.title,
       schema_version = excluded.schema_version,
       questions_json = excluded.questions_json,
       is_active = excluded.is_active,
       updated_by = excluded.updated_by,
       updated_at = excluded.updated_at`
  )
    .bind(
      crypto.randomUUID(),
      slug,
      title,
      str(parsed.schema_version || "1", 16),
      normalizedJSON,
      isActive,
      user.id,
      user.id,
      now,
      now
    )
    .run();
  return json({ ok: true, slug, updated_at: now });
}

async function adminBridgesGet(request: Request, env: Env): Promise<Response> {
  const user = await getWebUser(request, env);
  if (!user) {
    return json({ error: "login required", code: "unauthorized", login_url: `/auth/linuxdo/start?next=${encodeURIComponent("/admin/bridges")}` }, 401);
  }
  if (!isAdminUser(user, env)) return jsonError("forbidden", 403, "forbidden");
  const rows = await env.DB.prepare(
    `SELECT bridges.id, bridges.name, bridges.slug, bridges.icon_url, bridges.homepage_url, bridges.is_active, bridges.updated_at,
            bridge_base_urls.id AS base_url_id, bridge_base_urls.base_url, bridge_base_urls.host,
            bridge_base_urls.is_active AS base_url_active
     FROM bridges
     LEFT JOIN bridge_base_urls ON bridge_base_urls.bridge_id = bridges.id
     ORDER BY bridges.updated_at DESC, bridge_base_urls.base_url ASC`
  ).all();
  const byID = new Map<string, any>();
  for (const row of rows.results ?? []) {
    const r = row as any;
    if (!byID.has(r.id)) {
      byID.set(r.id, {
        id: r.id,
        name: r.name,
        slug: r.slug,
        icon_url: r.icon_url || "",
        homepage_url: r.homepage_url || "",
        is_active: !!r.is_active,
        updated_at: r.updated_at,
        base_urls: [],
      });
    }
    if (r.base_url_id) {
      byID.get(r.id).base_urls.push({
        id: r.base_url_id,
        base_url: r.base_url,
        host: r.host,
        is_active: !!r.base_url_active,
      });
    }
  }
  const suggestions = await env.DB.prepare(
    `SELECT bridge_suggestions.id, bridge_suggestions.base_url, bridge_suggestions.host, bridge_suggestions.source,
            bridge_suggestions.submitted_name, bridge_suggestions.page_title, bridge_suggestions.icon_url,
            bridge_suggestions.ai_name, bridge_suggestions.ai_slug, bridge_suggestions.ai_confidence,
            bridge_suggestions.ai_reason, bridge_suggestions.status, bridge_suggestions.occurrence_count,
            bridge_suggestions.created_at, bridge_suggestions.updated_at, bridge_suggestions.last_seen_at,
            users.username, users.login
     FROM bridge_suggestions
     LEFT JOIN users ON users.id = bridge_suggestions.user_id
     WHERE bridge_suggestions.status = 'pending'
     ORDER BY bridge_suggestions.occurrence_count DESC, bridge_suggestions.updated_at DESC
     LIMIT 50`
  ).all();
  return json({ user: publicUser(user), bridges: [...byID.values()], suggestions: suggestions.results ?? [] });
}

async function adminBridgesPost(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return jsonError("login required", 401, "unauthorized");
  if (!isAdminUser(user, env)) return jsonError("forbidden", 403, "forbidden");

  const body = await readJson<any>(request);
  const name = str(body.name, MAX_STRING_LENGTH).trim();
  if (!name) return jsonError("name is required", 400, "bad_request");
  const slug = slugify(str(body.slug, MAX_STRING_LENGTH).trim() || name) || `bridge-${crypto.randomUUID().slice(0, 8)}`;
  const isActive = body.is_active === false ? 0 : 1;
  const baseURLs = normalizeBridgeBaseURLs(body.base_urls);
  if (baseURLs.length < 1) return jsonError("at least one valid https base_url is required", 400, "bad_request");
  const iconURL = normalizePublicHTTPSURL(str(body.icon_url, MAX_URL_LENGTH));
  const homepageURL = normalizePublicHTTPSURL(str(body.homepage_url, MAX_URL_LENGTH));
  const suggestionID = str(body.suggestion_id, MAX_STRING_LENGTH);

  const conflict = await env.DB.prepare(
    `SELECT bridge_base_urls.base_url, bridges.name
     FROM bridge_base_urls
     JOIN bridges ON bridges.id = bridge_base_urls.bridge_id
     WHERE bridge_base_urls.base_url IN (${baseURLs.map(() => "?").join(",")})
       AND bridges.slug != ?`
  )
    .bind(...baseURLs.map((item) => item.baseURL), slug)
    .first<any>();
  if (conflict?.base_url) {
    return jsonError(`base_url already belongs to ${conflict.name}`, 409, "conflict");
  }

  const now = iso(new Date());
  const existing = await env.DB.prepare(`SELECT id FROM bridges WHERE slug = ?`).bind(slug).first<any>();
  const bridgeID = existing?.id || crypto.randomUUID();
  const statements = [
    env.DB.prepare(
      `INSERT INTO bridges (id, name, slug, icon_url, homepage_url, is_active, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?)
       ON CONFLICT(slug) DO UPDATE SET
         name = excluded.name,
         icon_url = excluded.icon_url,
         homepage_url = excluded.homepage_url,
         is_active = excluded.is_active,
         updated_at = excluded.updated_at`
    ).bind(bridgeID, name, slug, iconURL, homepageURL, isActive, now, now),
    env.DB.prepare(`UPDATE bridge_base_urls SET is_active = 0, updated_at = ? WHERE bridge_id = ?`).bind(now, bridgeID),
  ];
  for (const item of baseURLs) {
    statements.push(
      env.DB.prepare(
        `INSERT INTO bridge_base_urls (id, bridge_id, base_url, host, is_active, created_at, updated_at)
         VALUES (?, ?, ?, ?, 1, ?, ?)
         ON CONFLICT(base_url) DO UPDATE SET
           bridge_id = excluded.bridge_id,
           host = excluded.host,
           is_active = 1,
           updated_at = excluded.updated_at`
      ).bind(crypto.randomUUID(), bridgeID, item.baseURL, item.host, now, now)
    );
  }
  if (suggestionID) {
    statements.push(
      env.DB.prepare(`UPDATE bridge_suggestions SET status = 'approved', bridge_id = ?, updated_at = ? WHERE id = ?`).bind(bridgeID, now, suggestionID)
    );
  }
  await env.DB.batch(statements);
  return json({ ok: true, id: bridgeID, slug, updated_at: now });
}

async function adminBridgeIdentifyPost(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return jsonError("login required", 401, "unauthorized");
  if (!isAdminUser(user, env)) return jsonError("forbidden", 403, "forbidden");
  const body = await readJson<any>(request);
  const detected = await identifyBridgeCandidate(env, body.base_url, str(body.name || body.submitted_name, MAX_STRING_LENGTH));
  const suggestionID = str(body.suggestion_id, MAX_STRING_LENGTH);
  if (suggestionID) {
    await env.DB.prepare(
      `UPDATE bridge_suggestions
       SET page_title = ?, icon_url = ?, ai_name = ?, ai_slug = ?, ai_confidence = ?, ai_reason = ?, updated_at = ?
       WHERE id = ?`
    )
      .bind(detected.page_title, detected.icon_url, detected.detected_name, detected.slug, detected.confidence, detected.reason, iso(new Date()), suggestionID)
      .run();
  }
  return json({ ok: true, ...detected });
}

async function bridgeSuggestionPost(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return jsonError("login required", 401, "unauthorized");
  await enforceUserRateLimit(env, user.id, "bridge_suggestion", 20, 3600);
  const body = await readJson<any>(request);
  const baseURL = normalizeProviderBaseURL(body.base_url);
  if (!baseURL) return jsonError("base_url must be a valid https URL", 400, "bad_request");
  const suggestion = await upsertBridgeSuggestion(env, {
    userID: user.id,
    baseURL,
    host: hostFromProviderBaseURL(baseURL),
    source: "user",
    submittedName: str(body.name || body.submitted_name, MAX_STRING_LENGTH),
  });
  return json({ ok: true, suggestion });
}

async function loadEditableQuestionBank(env: Env): Promise<any | null> {
  const defaultRow = await env.DB.prepare(
    `SELECT slug, title, schema_version, questions_json, is_active, updated_at
     FROM question_banks WHERE slug = ?`
  )
    .bind(DEFAULT_QUESTION_BANK_SLUG)
    .first<any>();
  if (defaultRow) return defaultRow;
  return env.DB.prepare(
    `SELECT slug, title, schema_version, questions_json, is_active, updated_at
     FROM question_banks WHERE is_active = 1 ORDER BY updated_at DESC LIMIT 1`
  ).first<any>();
}

async function adminPage(request: Request, env: Env): Promise<Response> {
  const user = await getWebUser(request, env);
  const loginURL = `/auth/linuxdo/start?next=${encodeURIComponent("/admin")}`;
  if (!user) {
    return html(layoutPage("LD-gpt-check 管理后台", `
      <section class="hero">
        <span class="badge">Admin</span>
        <h1>管理后台</h1>
        <p>管理员登录后可以维护题库和检查公开接口状态。</p>
        <div class="login-actions">
          <a class="linuxdo-button" href="${loginURL}" aria-label="使用 Linux.do 登录">
            ${linuxdoIcon()}
            <span><strong>使用 Linux.do 登录</strong><small>只有管理员账号会显示管理功能</small></span>
          </a>
        </div>
      </section>
    `), 401);
  }
  if (!isAdminUser(user, env)) {
    return html(resultPage("无权访问", "当前 Linux.do 账号不在管理员列表中。"), 403);
  }
  const publicBase = str(env.BASE_URL || "", MAX_STRING_LENGTH).replace(/\/$/, "");
  return html(layoutPage("LD-gpt-check 管理后台", `
    <section class="hero">
      <span class="badge">Admin</span>
      ${userIdentityBlock(user)}
      <p>这里是管理入口。管理员功能集中放在同源 Worker 内，生产前端和后端共用同一个部署。</p>
      <div class="actions">
        <a class="button" href="/admin/questions">题目 JSON 管理</a>
        <a class="button" href="/admin/bridges">中转站映射管理</a>
        <a class="button secondary" href="/account">返回账号</a>
      </div>
    </section>
    <section class="grid">
      <article><span>管理员 UID</span><strong>${escapeHTML(str(user.provider_user_id || user.id, MAX_STRING_LENGTH))}</strong></article>
      <article><span>公开题库</span><strong><a class="inline-link" href="/api/v1/questions" target="_blank" rel="noreferrer">/api/v1/questions</a></strong></article>
      <article><span>Worker</span><strong>${escapeHTML(publicBase || "当前同源")}</strong></article>
    </section>
    <section class="panel">
      <h2>管理模块</h2>
      <div class="admin-list">
        <a href="/admin/questions">
          <strong>题目 JSON 管理</strong>
          <span>编辑远程题库，CLI 默认从公开题库接口拉取。</span>
        </a>
        <a href="/admin/bridges">
          <strong>中转站映射管理</strong>
          <span>配置全局 base URL 到中转站名称的对应关系。</span>
        </a>
      </div>
    </section>
  `));
}

async function deviceStart(request: Request, env: Env): Promise<Response> {
  await enforceRateLimit(request, env, "device_start", 20, 600);
  const now = new Date();
  const deviceCode = await randomToken("dc");
  const userCode = numericCode();
  const id = crypto.randomUUID();
  await env.DB.prepare(
    `INSERT INTO device_sessions
      (id, device_code_hash, user_code_hash, status, expires_at, created_at)
     VALUES (?, ?, ?, 'pending', ?, ?)`
  )
    .bind(
      id,
      await hashSecret(deviceCode, env),
      await hashSecret(normalizeCode(userCode), env),
      iso(addSeconds(now, DEVICE_EXPIRES_SECONDS)),
      iso(now)
    )
    .run();

  const verification = `${env.BASE_URL.replace(/\/$/, "")}/device`;
  return json({
    device_code: deviceCode,
    user_code: userCode,
    verification_uri: verification,
    verification_uri_complete: `${verification}?code=${encodeURIComponent(userCode)}`,
    expires_in: DEVICE_EXPIRES_SECONDS,
    interval: POLL_INTERVAL_SECONDS,
  });
}

async function devicePoll(request: Request, env: Env): Promise<Response> {
  await enforceRateLimit(request, env, "device_poll", 120, 60);
  const body = await readJson<{ device_code?: string }>(request);
  if (!body.device_code) return jsonError("device_code is required", 400, "bad_request");

  const now = new Date();
  const row = await env.DB.prepare(`SELECT * FROM device_sessions WHERE device_code_hash = ?`)
    .bind(await hashSecret(body.device_code, env))
    .first<any>();
  if (!row) return jsonError("invalid device_code", 400, "bad_request");
  if (row.status === "consumed") return json({ status: "expired" });
  if (row.expires_at <= iso(now)) {
    await env.DB.prepare(`UPDATE device_sessions SET status = 'expired' WHERE id = ?`).bind(row.id).run();
    return json({ status: "expired" });
  }
  if (row.last_polled_at && secondsBetween(row.last_polled_at, now) < POLL_INTERVAL_SECONDS) {
    return json({ status: "slow_down" });
  }
  await env.DB.prepare(`UPDATE device_sessions SET last_polled_at = ? WHERE id = ?`).bind(iso(now), row.id).run();

  if (row.status === "pending") return json({ status: "pending" });
  if (row.status !== "approved" || !row.user_id) return json({ status: "expired" });

  const consumed = await env.DB.prepare(
    `UPDATE device_sessions SET status = 'consumed'
     WHERE id = ? AND status = 'approved'`
  )
    .bind(row.id)
    .run();
  if ((consumed.meta?.changes ?? 0) !== 1) return json({ status: "expired" });

  const token = await randomToken("ldgc");
  const tokenID = crypto.randomUUID();
  await env.DB.prepare(
    `INSERT INTO access_tokens (id, user_id, token_hash, device_name, created_at, expires_at)
     VALUES (?, ?, ?, 'CLI device login', ?, ?)`
  )
    .bind(tokenID, row.user_id, await hashSecret(token, env), iso(now), iso(addSeconds(now, ACCESS_TOKEN_EXPIRES_SECONDS)))
    .run();

  const user = await env.DB.prepare(`SELECT id, username FROM users WHERE id = ?`).bind(row.user_id).first<any>();
  return json({
    status: "authorized",
    access_token: token,
    user: { id: user?.id ?? row.user_id, username: user?.username ?? "" },
  });
}

async function devicePage(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const hasCode = !!url.searchParams.get("code");
  const user = await getWebUser(request, env);
  const loginURL = `/auth/linuxdo/start?next=${encodeURIComponent("/device")}`;
  const turnstileEnabled = !!(env.TURNSTILE_SITE_KEY && env.TURNSTILE_SECRET_KEY);
  const scriptNonce = cspNonce();
  const turnstile = turnstileEnabled
    ? `<div class="turnstile"><div class="cf-turnstile" data-sitekey="${escapeHTML(env.TURNSTILE_SITE_KEY || "")}" data-callback="ldgcTurnstileOK" data-expired-callback="ldgcTurnstileReset" data-error-callback="ldgcTurnstileReset"></div><p id="turnstile-status" class="status">完成人机验证后可授权。</p></div>`
    : "";
  const pageScript = `<script nonce="${scriptNonce}">
document.addEventListener("DOMContentLoaded", function () {
  var inputs = Array.prototype.slice.call(document.querySelectorAll("input[name='code_digit']"));
  var autofill = document.getElementById("otp-autofill");
  function clean(value) {
    return String(value || "").replace(/\\D/g, "").slice(0, 9);
  }
  function fillOTP(value) {
    var text = clean(value);
    if (!text) return;
    inputs.forEach(function (box, i) { box.value = text.slice(i * 3, i * 3 + 3); });
    var next = inputs.find(function (box) { return box.value.length < 3; });
    (next || inputs[inputs.length - 1]).focus();
  }
  inputs.forEach(function (input, index) {
    input.addEventListener("input", function () {
      var text = clean(input.value);
      if (text.length > 3) {
        fillOTP(text);
        return;
      }
      input.value = text;
      if (input.value.length === 3 && inputs[index + 1]) inputs[index + 1].focus();
    });
    input.addEventListener("keydown", function (event) {
      if (event.key === "Backspace" && input.value === "" && inputs[index - 1]) inputs[index - 1].focus();
    });
    input.addEventListener("paste", function (event) {
      var clipboard = event.clipboardData || window.clipboardData;
      if (!clipboard) return;
      var text = clean(clipboard.getData("text"));
      if (text.length < 4) return;
      event.preventDefault();
      fillOTP(text);
    });
  });
  if (autofill) {
    autofill.addEventListener("input", function () { fillOTP(autofill.value); });
    autofill.addEventListener("change", function () { fillOTP(autofill.value); });
  }
  var form = document.getElementById("approve-form");
  if (form) {
    form.addEventListener("submit", function () {
      var joined = inputs.map(function (box) { return box.value; }).join("");
      if (clean(joined).length === 9) fillOTP(joined);
    });
  }
});
${turnstileEnabled ? `
window.ldgcTurnstileOK = function () {
  var button = document.getElementById("approve-submit");
  var status = document.getElementById("turnstile-status");
  if (button) button.disabled = false;
  if (status) status.textContent = "验证完成，可以授权。";
};
window.ldgcTurnstileReset = function () {
  var button = document.getElementById("approve-submit");
  var status = document.getElementById("turnstile-status");
  if (button) button.disabled = true;
  if (status) status.textContent = "请先完成人机验证。";
};
` : ""}
</script>${turnstileEnabled ? `<script nonce="${scriptNonce}" src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>` : ""}`;
  const otpInputs = Array.from({ length: 3 }, (_, i) =>
    `<input class="otp-box" name="code_digit" aria-label="验证码第 ${i + 1} 组" inputmode="numeric" autocomplete="${i === 0 ? "one-time-code" : "off"}" maxlength="${i === 0 ? "11" : "3"}" pattern="${i === 0 ? "[0-9 -]{3,11}" : "[0-9]{3}"}" placeholder="000" required>`
  ).join("");
  const body = `<!doctype html>
<html lang="zh-CN">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>LD-gpt-check 登录</title>
<style>
:root{color-scheme:light;--text:#0f172a;--muted:#64748b;--line:#dbeafe;--brand:#2563eb;--brand2:#06b6d4;--bg:#f7fbff}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:var(--text);background:linear-gradient(135deg,rgba(37,99,235,.1),transparent 34%),linear-gradient(225deg,rgba(6,182,212,.14),transparent 38%),var(--bg);display:grid;place-items:center;padding:24px;line-height:1.5}
.shell{width:min(100%,560px);border:1px solid rgba(191,219,254,.9);background:rgba(255,255,255,.88);box-shadow:0 24px 80px rgba(37,99,235,.16);backdrop-filter:blur(18px);border-radius:16px;overflow:hidden}
.top{padding:22px 24px;border-bottom:1px solid #e2e8f0;background:rgba(255,255,255,.72)}.brand{font-weight:750;letter-spacing:0}.badge{display:inline-flex;margin-bottom:12px;border:1px solid #bfdbfe;border-radius:999px;padding:5px 10px;font:12px ui-monospace,SFMono-Regular,Menlo,monospace;color:#1d4ed8;background:#eff6ff}
.content{padding:24px}h1{margin:0;font-size:28px;line-height:1.15;letter-spacing:0}p{margin:12px 0 0;color:var(--muted)}.user{margin-top:18px;padding:12px;border:1px solid #e2e8f0;border-radius:10px;background:#f8fafc;color:#334155}
label{display:block;margin-top:20px;font-weight:650}.otp{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:10px}.otp-box{width:100%;min-height:58px;border:1px solid #cbd5e1;border-radius:10px;text-align:center;font:700 24px ui-monospace,SFMono-Regular,Menlo,monospace;letter-spacing:.08em;color:var(--text);background:#fff}.otp-box::placeholder{color:#cbd5e1}.otp-box:focus{outline:3px solid rgba(37,99,235,.18);border-color:var(--brand)}.otp-autofill{position:absolute;left:-9999px;width:1px;height:1px;opacity:0}
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:20px}button,a.button{appearance:none;border:1px solid var(--brand);border-radius:10px;background:linear-gradient(135deg,var(--brand),var(--brand2));color:#fff;min-height:44px;padding:10px 14px;font:700 15px system-ui,sans-serif;text-decoration:none;display:inline-flex;align-items:center;justify-content:center;cursor:pointer}button:disabled{opacity:.55;cursor:not-allowed;filter:grayscale(.35)}button.secondary,a.secondary{border-color:#cbd5e1;background:#fff;color:#334155}
.turnstile{margin-top:18px}.status{font-size:13px}.hint{margin-top:18px;border-top:1px solid #e2e8f0;padding-top:16px;font-size:14px}.code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#1d4ed8}
@media(max-width:520px){body{padding:12px}.content,.top{padding:18px}h1{font-size:24px}.otp{gap:8px}.otp-box{font-size:20px}button,a.button{width:100%}}
</style></head>
<body>
<main class="shell">
<div class="top"><span class="badge">CLI device authorization</span><div class="brand">LD-gpt-check</div></div>
<section class="content">
<h1>授权命令行设备</h1>
${user ? `<p class="user">当前登录用户：<strong>${escapeHTML(user.username || user.id)}</strong></p>
<form id="approve-form" method="post" action="/api/device/approve">
  <label>输入终端显示的 9 位验证码</label>
  <input id="otp-autofill" class="otp-autofill" type="text" inputmode="numeric" autocomplete="one-time-code" tabindex="-1" aria-hidden="true">
  <div class="otp">${otpInputs}</div>
  ${turnstile}
  <div class="actions"><button id="approve-submit" type="submit"${turnstileEnabled ? " disabled" : ""}>授权 CLI</button><button class="secondary" type="submit" form="logout-form">退出登录</button></div>
</form>
<form id="logout-form" method="post" action="/logout"></form>
<p class="hint">确认验证码和终端中的 <span class="code">user_code</span> 一致后再授权。</p>` : `<p>请先使用 Linux.do 登录，然后回到这里输入终端显示的 9 位验证码。</p><div class="actions"><a class="button" href="${loginURL}">使用 Linux.do 登录</a></div><p class="hint">${hasCode ? "为避免误授权，页面不会自动填入验证码；请从终端复制后手动输入。" : "如果你在 SSH、WSL 或远程服务器上运行 CLI，可以复制终端打印的链接到浏览器打开。"}</p>`}
</section>
</main>
${pageScript}
</body></html>`;
  return html(body, 200, scriptNonce);
}

async function accountPage(request: Request, env: Env): Promise<Response> {
  const user = await getWebUser(request, env);
  const loginURL = `/auth/linuxdo/start?next=${encodeURIComponent("/account")}`;
  if (!user) {
    return html(layoutPage("LD-gpt-check 账号", `
      <section class="hero">
        <span class="badge">Linux.do OAuth</span>
        <h1>登录 LD-gpt-check</h1>
        <p>登录后可以授权 CLI 设备，查看账号状态和最近上传记录。</p>
        <div class="login-actions">
          <a class="linuxdo-button" href="${loginURL}" aria-label="使用 Linux.do 登录">
            ${linuxdoIcon()}
            <span><strong>使用 Linux.do 登录</strong><small>OAuth 授权，不会把密码交给本站</small></span>
          </a>
        </div>
      </section>
    `));
  }

  const stats = await env.DB.prepare(
    `SELECT
       COUNT(*) AS total_submissions,
       MAX(created_at) AS last_submission_at,
       AVG(accuracy) AS avg_accuracy
     FROM benchmark_submissions WHERE user_id = ?`
  )
    .bind(user.id)
    .first<any>();
  const recent = await env.DB.prepare(
    `SELECT benchmark_submissions.id, model, reasoning_effort, attempt_count, correct_count, accuracy, is_anonymous,
            codex_channel, bridges.name AS codex_bridge_name, benchmark_submissions.created_at
     FROM benchmark_submissions
     LEFT JOIN bridges ON bridges.id = benchmark_submissions.codex_bridge_id
     WHERE user_id = ? ORDER BY benchmark_submissions.created_at DESC LIMIT 10`
  )
    .bind(user.id)
    .all<any>();

  const rows = (recent.results ?? [])
    .map(
      (r: any) => `<tr>
        <td>${escapeHTML(str(r.model || "-", 80))}</td>
        <td>${escapeHTML(str(r.reasoning_effort || "-", 32))}</td>
        <td>${int(r.correct_count)}/${int(r.attempt_count)}</td>
        <td>${formatPercent(num(r.accuracy))}</td>
        <td>${channelLabel(r.codex_channel, r.codex_bridge_name)}</td>
        <td>${r.is_anonymous ? "匿名" : "公开"}</td>
        <td>${escapeHTML(formatDate(r.created_at))}</td>
        <td>
          <form method="post" action="/account/submissions/delete">
            <input type="hidden" name="submission_id" value="${escapeHTML(str(r.id, MAX_STRING_LENGTH))}">
            <button class="danger small" type="submit">删除</button>
          </form>
        </td>
      </tr>`
    )
    .join("");
  const adminActions = isAdminUser(user, env)
    ? `<a class="button secondary" href="/admin">管理后台</a>`
    : "";

  return html(layoutPage("LD-gpt-check 账号", `
    <section class="hero">
      <span class="badge">已登录</span>
      ${userIdentityBlock(user)}
      <p>当前网页会话已通过 Linux.do 登录。你可以继续授权 CLI，或退出当前浏览器登录。</p>
      <div class="actions">
        ${adminActions}
        <a class="button" href="/device">授权 CLI 设备</a>
        <form method="post" action="/logout"><button class="secondary" type="submit">退出登录</button></form>
      </div>
    </section>
    <section class="grid">
      <article><span>用户 ID</span><strong>${escapeHTML(user.id)}</strong></article>
      <article><span>累计上传</span><strong>${int(stats?.total_submissions)} 次</strong></article>
      <article><span>平均正确率</span><strong>${stats?.avg_accuracy == null ? "-" : formatPercent(num(stats.avg_accuracy))}</strong></article>
    </section>
    <section class="panel">
      <h2>CLI 配置</h2>
      <p>在本机终端使用下面的 API 地址登录。</p>
      <pre>LD_GPT_CHECK_API_BASE_URL=${escapeHTML(env.BASE_URL.replace(/\/$/, ""))} bin/ld-gpt-check login</pre>
    </section>
    <section class="panel">
      <h2>提交中转站</h2>
      <p>如果你的 Codex 使用了中转站，可以提交 provider base URL。管理员审核后，后续上传会显示对应中转站标签。</p>
      <form method="post" action="/account/bridge-suggestions">
        <div class="actions">
          <input name="name" maxlength="128" placeholder="中转站名称，可选">
          <input name="base_url" maxlength="256" placeholder="https://bridge.example.com/v1" required>
          <button type="submit">提交候选</button>
        </div>
      </form>
    </section>
    <section class="panel">
      <h2>最近上传</h2>
      <p>这里只展示最后 10 条记录。删除操作只会删除你的测试数据，不会删除账号、网页会话或 CLI token。</p>
      ${rows ? `<div class="table-wrap"><table><thead><tr><th>模型</th><th>推理</th><th>正确</th><th>正确率</th><th>渠道</th><th>展示</th><th>时间</th><th>操作</th></tr></thead><tbody>${rows}</tbody></table></div>` : `<p>还没有上传记录。</p>`}
      ${rows ? `<form class="delete-all" method="post" action="/account/submissions/delete-all">
        <label>清空全部测试数据：输入 <strong>DELETE</strong> 确认</label>
        <div class="actions"><input name="confirm" autocomplete="off" placeholder="DELETE"><button class="danger" type="submit">清空我的测试数据</button></div>
      </form>` : ""}
    </section>
  `));
}

async function deviceApprove(request: Request, env: Env): Promise<Response> {
  const wantsHTML = (request.headers.get("content-type") || "").includes("application/x-www-form-urlencoded");
  try {
    enforceBodySize(request);
    await enforceRateLimit(request, env, "device_approve", 30, 600);
    enforceSameOrigin(request, env);
  } catch (err) {
    if (wantsHTML && err instanceof APIError) return html(resultPage("请求被拒绝", err.publicMessage), err.status);
    throw err;
  }
  const user = await getWebUser(request, env);
  if (!user) return wantsHTML ? html(resultPage("需要登录", "请先使用 Linux.do 登录后再授权 CLI。"), 401) : jsonError("login required", 401, "unauthorized");

  let input: Record<string, any>;
  if (wantsHTML) {
    const form = await request.formData();
    input = Object.fromEntries(form);
    const digits = form.getAll("code_digit").map((v) => String(v)).join("");
    if (digits) input.user_code = digits;
  } else {
    input = await readJson<any>(request);
  }
  try {
    await verifyTurnstileIfConfigured(request, env, input);
  } catch (err) {
    if (wantsHTML && err instanceof APIError) return html(resultPage("验证失败", err.publicMessage), err.status);
    throw err;
  }
  const code = normalizeCode(String(input.user_code || ""));
  if (!code) return wantsHTML ? html(resultPage("验证码缺失", "请回到终端复制完整验证码后重试。"), 400) : jsonError("user_code is required", 400, "bad_request");

  const now = new Date();
  const row = await env.DB.prepare(
    `SELECT id, expires_at FROM device_sessions
     WHERE user_code_hash = ? AND status = 'pending'`
  )
    .bind(await hashSecret(code, env))
    .first<any>();
  if (!row || row.expires_at <= iso(now)) {
    return wantsHTML
      ? html(resultPage("验证码无效或已过期", "请回到终端重新执行登录命令，生成新的验证码。"), 400)
      : jsonError("device code is invalid or expired", 400, "bad_request");
  }

  await env.DB.prepare(
    `UPDATE device_sessions SET status = 'approved', user_id = ?, approved_at = ? WHERE id = ?`
  )
    .bind(user.id, iso(now), row.id)
    .run();

  if (wantsHTML) return html(resultPage("已授权", "可以回到终端继续。"));
  return json({ ok: true });
}

async function oauthStart(request: Request, env: Env): Promise<Response> {
  await enforceRateLimit(request, env, "oauth_start", 20, 600);
  const url = new URL(request.url);
  const redirectPath = safeRedirect(url.searchParams.get("next") || "/account");
  const state = await randomToken("st");
  const now = new Date();
  await env.DB.prepare(
    `INSERT INTO oauth_states (id, state_hash, redirect_path, expires_at, created_at)
     VALUES (?, ?, ?, ?, ?)`
  )
    .bind(crypto.randomUUID(), await hashSecret(state, env), redirectPath, iso(addSeconds(now, 600)), iso(now))
    .run();

  const authURL = new URL(env.LINUXDO_AUTH_URL);
  authURL.searchParams.set("response_type", "code");
  authURL.searchParams.set("client_id", env.LINUXDO_CLIENT_ID);
  authURL.searchParams.set("redirect_uri", `${env.BASE_URL.replace(/\/$/, "")}/auth/linuxdo/callback`);
  authURL.searchParams.set("state", state);
  return redirect(authURL.toString(), cookie("ldgc_oauth_state", state, env, 600));
}

async function oauthCallback(request: Request, env: Env): Promise<Response> {
  await enforceRateLimit(request, env, "oauth_callback", 60, 600);
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  const cookieState = parseCookies(request.headers.get("cookie")).ldgc_oauth_state;
  if (!code || !state || state !== cookieState) return jsonError("invalid oauth state", 400, "bad_request");

  const stateHash = await hashSecret(state, env);
  const stateRow = await env.DB.prepare(`SELECT * FROM oauth_states WHERE state_hash = ? AND used_at IS NULL`)
    .bind(stateHash)
    .first<any>();
  if (!stateRow || stateRow.expires_at <= iso(new Date())) return jsonError("oauth state expired", 400, "bad_request");
  await env.DB.prepare(`UPDATE oauth_states SET used_at = ? WHERE id = ?`).bind(iso(new Date()), stateRow.id).run();

  const tokenResp = await fetch(env.LINUXDO_TOKEN_URL, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: `${env.BASE_URL.replace(/\/$/, "")}/auth/linuxdo/callback`,
      client_id: env.LINUXDO_CLIENT_ID,
      client_secret: env.LINUXDO_CLIENT_SECRET,
    }),
  });
  if (!tokenResp.ok) return jsonError("oauth token exchange failed", 502, "upstream_error");
  const tokenJSON: any = await tokenResp.json();
  const oauthToken = tokenJSON.access_token;
  if (!oauthToken) return jsonError("oauth access_token missing", 502, "upstream_error");

  const userResp = await fetch(env.LINUXDO_USERINFO_URL, {
    headers: { authorization: `Bearer ${oauthToken}` },
  });
  if (!userResp.ok) return jsonError("oauth userinfo failed", 502, "upstream_error");
  const profile: any = await userResp.json();
  const user = await upsertLinuxDoUser(profile, env);

  const sessionToken = await randomToken("ws");
  await env.DB.prepare(
    `INSERT INTO web_sessions (id, user_id, session_hash, created_at, expires_at)
     VALUES (?, ?, ?, ?, ?)`
  )
    .bind(crypto.randomUUID(), user.id, await hashSecret(sessionToken, env), iso(new Date()), iso(addSeconds(new Date(), 30 * 86400)))
    .run();
  return redirect(stateRow.redirect_path || "/device", [
    cookie("ldgc_session", sessionToken, env, 30 * 86400),
    cookie("ldgc_oauth_state", "", env, 0),
  ]);
}

async function apiMe(request: Request, env: Env): Promise<Response> {
  const auth = await getBearerUser(request, env);
  if (!auth) return jsonError("unauthorized", 401, "unauthorized");
  return json({ user: auth.user });
}

async function createSubmission(request: Request, env: Env): Promise<Response> {
  const auth = await getBearerUser(request, env);
  if (!auth) return jsonError("unauthorized", 401, "unauthorized");
  await enforceUserRateLimit(env, auth.user.id, "create_submission", 60, 3600);
  const p = await readJson<any>(request);
  const validationError = validateSubmissionPayload(p);
  if (validationError) return jsonError(validationError, 422, "validation_failed");

  const existing = await env.DB.prepare(
    `SELECT id FROM benchmark_submissions WHERE user_id = ? AND upload_id = ?`
  )
    .bind(auth.user.id, str(p.upload_id, MAX_STRING_LENGTH))
    .first<any>();
  if (existing?.id) return json({ id: existing.id, duplicate: true });

  const now = iso(new Date());
  const submissionID = crypto.randomUUID();
  const questions = p.questions.slice(0, MAX_QUESTIONS);
  const attempts = p.attempts.slice(0, MAX_ATTEMPTS);
  const provider = await classifyProviderBaseURL(env, String(p.codex_provider_base_url || ""));
  if (provider.channel === "unknown_bridge") {
    await upsertUnknownProviderSuggestion(env, auth.user.id, provider);
  }
  const statements = [
    env.DB.prepare(
      `INSERT INTO benchmark_submissions
       (id, user_id, upload_id, upload_schema_version, client_version, model, reasoning_effort, question_count,
        attempt_count, correct_count, accuracy,
        avg_input_tokens, avg_output_tokens, avg_reason_tokens, avg_time_seconds, avg_tps, is_anonymous,
        started_at, finished_at, duration_seconds, question_suite, client_timezone,
        os, arch, codex_version, codex_model_source, codex_model_provider, codex_provider_host,
        codex_provider_base_url, codex_channel, codex_bridge_id,
        codex_sandbox, codex_ephemeral, codex_skip_git_repo_check, codex_disabled_features, codex_invocation, created_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    ).bind(
      submissionID,
      auth.user.id,
      str(p.upload_id, MAX_STRING_LENGTH),
      int(p.upload_schema_version || 1),
      str(p.client_version, MAX_STRING_LENGTH),
      str(p.model, MAX_STRING_LENGTH),
      str(p.reasoning_effort, MAX_STRING_LENGTH),
      int(p.question_count),
      int(p.attempt_count),
      int(p.correct),
      num(p.accuracy),
      num(p.avg_input_tokens),
      num(p.avg_output_tokens),
      num(p.avg_reason_tokens),
      num(p.avg_time_seconds),
      num(p.avg_tps),
      p.anonymous ? 1 : 0,
      str(p.started_at, MAX_STRING_LENGTH),
      str(p.finished_at, MAX_STRING_LENGTH),
      num(p.duration_seconds),
      str(p.question_suite, MAX_STRING_LENGTH),
      str(p.client_timezone, 32),
      str(p.os, MAX_STRING_LENGTH),
      str(p.arch, MAX_STRING_LENGTH),
      str(p.codex_version, MAX_STRING_LENGTH),
      str(p.codex_model_source, 32),
      str(p.codex_model_provider, MAX_STRING_LENGTH),
      provider.host || str(p.codex_provider_host, MAX_STRING_LENGTH),
      provider.baseURL,
      provider.channel,
      provider.bridgeID,
      str(p.codex_sandbox, 32),
      p.codex_ephemeral ? 1 : 0,
      p.codex_skip_git_repo_check ? 1 : 0,
      jsonArrayString(p.codex_disabled_features, MAX_METADATA_LENGTH),
      str(p.codex_invocation, MAX_METADATA_LENGTH),
      now
    ),
  ];
  for (const q of questions) {
    statements.push(
      env.DB.prepare(
        `INSERT INTO benchmark_question_results
         (id, submission_id, question_id, question_version, question_title, grader_type,
          expected_answer, prompt_hash, test_count, correct_count, accuracy,
          avg_input_tokens, avg_output_tokens, avg_reason_tokens, avg_time_seconds, avg_tps, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      ).bind(
        crypto.randomUUID(),
        submissionID,
        str(q.question_id, MAX_STRING_LENGTH),
        str(q.question_version, MAX_STRING_LENGTH),
        str(q.question_title, MAX_STRING_LENGTH),
        str(q.grader_type, 32),
        str(q.expected_answer, MAX_STRING_LENGTH),
        str(q.prompt_hash, 64),
        int(q.tests),
        int(q.correct),
        num(q.accuracy),
        num(q.avg_input_tokens),
        num(q.avg_output_tokens),
        num(q.avg_reason_tokens),
        num(q.avg_time_seconds),
        num(q.avg_tps),
        now
      )
    );
  }
  for (const a of attempts) {
    statements.push(
      env.DB.prepare(
        `INSERT INTO benchmark_attempts
         (id, submission_id, question_id, question_version, case_index, status, is_correct,
          expected_answer, extracted_answer, failure_reason, answer_preview, answer_preview_truncated, answer_hash,
          input_tokens, cached_input_tokens, output_tokens, reasoning_tokens, total_tokens, time_seconds, tps,
          codex_thread_id, event_count, event_types, tool_event_detected, answer_chars,
          error_code, started_at, finished_at, timeout_seconds, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      ).bind(
        crypto.randomUUID(),
        submissionID,
        str(a.question_id, MAX_STRING_LENGTH),
        str(a.question_version, MAX_STRING_LENGTH),
        int(a.case_index),
        str(a.status, 32),
        a.is_correct ? 1 : 0,
        str(a.expected_answer, MAX_STRING_LENGTH),
        str(a.extracted_answer, MAX_STRING_LENGTH),
        str(a.failure_reason, 64),
        str(a.answer_preview, MAX_PREVIEW_LENGTH),
        a.answer_preview_truncated ? 1 : 0,
        str(a.answer_hash, 64),
        int(a.input_tokens),
        int(a.cached_input_tokens),
        int(a.output_tokens),
        int(a.reasoning_tokens),
        int(a.total_tokens),
        num(a.time_seconds),
        num(a.tps),
        str(a.codex_thread_id, MAX_STRING_LENGTH),
        int(a.event_count),
        jsonArrayString(a.event_types, MAX_METADATA_LENGTH),
        a.tool_event_detected ? 1 : 0,
        int(a.answer_chars),
        str(a.error_code, 64),
        str(a.started_at, MAX_STRING_LENGTH),
        str(a.finished_at, MAX_STRING_LENGTH),
        num(a.timeout_seconds),
        now
      )
    );
  }
  try {
    await env.DB.batch(statements);
  } catch (err) {
    const duplicate = await env.DB.prepare(
      `SELECT id FROM benchmark_submissions WHERE user_id = ? AND upload_id = ?`
    )
      .bind(auth.user.id, str(p.upload_id, MAX_STRING_LENGTH))
      .first<any>();
    if (duplicate?.id) return json({ id: duplicate.id, duplicate: true });
    throw err;
  }
  return json({ id: submissionID, duplicate: false });
}

async function listSubmissions(request: Request, env: Env): Promise<Response> {
  const auth = await getBearerUser(request, env);
  if (!auth) return jsonError("unauthorized", 401, "unauthorized");
  const url = new URL(request.url);
  const limit = clampInt(url.searchParams.get("limit"), 1, 100, 50);
  const rows = await env.DB.prepare(
    `SELECT id, upload_id, model, reasoning_effort, question_count, attempt_count, correct_count,
            accuracy, avg_time_seconds, avg_tps, is_anonymous,
            codex_provider_base_url, codex_channel, codex_bridge_id, bridges.name AS codex_bridge_name,
            benchmark_submissions.created_at
     FROM benchmark_submissions
     LEFT JOIN bridges ON bridges.id = benchmark_submissions.codex_bridge_id
     WHERE user_id = ? ORDER BY benchmark_submissions.created_at DESC LIMIT ?`
  )
    .bind(auth.user.id, limit)
    .all();
  return json({
    submissions: (rows.results ?? []).map((row: any) => ({
      ...row,
      anonymous: !!row.is_anonymous,
      user: submissionDisplayUser(auth.user, !!row.is_anonymous),
    })),
  });
}

async function deleteSubmission(request: Request, env: Env): Promise<Response> {
  const auth = await getBearerUser(request, env);
  if (!auth) return jsonError("unauthorized", 401, "unauthorized");
  await enforceUserRateLimit(env, auth.user.id, "delete_submission", 120, 3600);
  const id = decodeURIComponent(new URL(request.url).pathname.split("/").pop() || "");
  if (!id) return jsonError("submission id is required", 400, "bad_request");
  const deleted = await deleteOwnSubmission(env, auth.user.id, id);
  return json({ ok: true, deleted });
}

async function deleteAllSubmissions(request: Request, env: Env): Promise<Response> {
  const auth = await getBearerUser(request, env);
  if (!auth) return jsonError("unauthorized", 401, "unauthorized");
  await enforceUserRateLimit(env, auth.user.id, "delete_all_submissions", 20, 3600);
  const deleted = await deleteOwnSubmissions(env, auth.user.id);
  return json({ ok: true, deleted });
}

async function apiLogout(request: Request, env: Env): Promise<Response> {
  const auth = await getBearerUser(request, env);
  if (auth) {
    await env.DB.prepare(`UPDATE access_tokens SET revoked_at = ? WHERE id = ?`).bind(iso(new Date()), auth.tokenID).run();
  }
  return json({ ok: true });
}

async function webLogout(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const token = parseCookies(request.headers.get("cookie")).ldgc_session;
  if (token) {
    await env.DB.prepare(`UPDATE web_sessions SET revoked_at = ? WHERE session_hash = ?`)
      .bind(iso(new Date()), await hashSecret(token, env))
      .run();
  }
  return redirect("/account", cookie("ldgc_session", "", env, 0));
}

async function webDeleteSubmission(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return html(resultPage("需要登录", "请先登录后再删除测试数据。"), 401);
  const form = await request.formData();
  const id = String(form.get("submission_id") || "");
  if (!id) return html(resultPage("缺少记录 ID", "请回到账号页重试。"), 400);
  await deleteOwnSubmission(env, user.id, id);
  return redirect("/account");
}

async function webDeleteAllSubmissions(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return html(resultPage("需要登录", "请先登录后再删除测试数据。"), 401);
  const form = await request.formData();
  if (String(form.get("confirm") || "") !== "DELETE") {
    return html(resultPage("确认文本不匹配", "请输入 DELETE 后再清空全部测试数据。"), 400);
  }
  await deleteOwnSubmissions(env, user.id);
  return redirect("/account");
}

async function webBridgeSuggestionPost(request: Request, env: Env): Promise<Response> {
  enforceSameOrigin(request, env);
  const user = await getWebUser(request, env);
  if (!user) return html(resultPage("需要登录", "请先登录后再提交中转站。"), 401);
  await enforceUserRateLimit(env, user.id, "bridge_suggestion", 20, 3600);
  enforceBodySize(request);
  const form = await request.formData();
  const baseURL = normalizeProviderBaseURL(String(form.get("base_url") || ""));
  if (!baseURL) return html(resultPage("中转站地址无效", "请提交 HTTPS base URL，例如 https://bridge.example.com/v1。"), 400);
  await upsertBridgeSuggestion(env, {
    userID: user.id,
    baseURL,
    host: hostFromProviderBaseURL(baseURL),
    source: "user",
    submittedName: str(String(form.get("name") || ""), MAX_STRING_LENGTH),
  });
  return html(resultPage("已提交中转站", "候选中转站已进入管理员审核队列。"), 200);
}

async function deleteOwnSubmission(env: Env, userID: string, submissionID: string): Promise<number> {
  const row = await env.DB.prepare(`SELECT id FROM benchmark_submissions WHERE id = ? AND user_id = ?`)
    .bind(submissionID, userID)
    .first<any>();
  if (!row?.id) return 0;
  await env.DB.batch([
    env.DB.prepare(`DELETE FROM benchmark_attempts WHERE submission_id = ?`).bind(submissionID),
    env.DB.prepare(`DELETE FROM benchmark_question_results WHERE submission_id = ?`).bind(submissionID),
    env.DB.prepare(`DELETE FROM benchmark_submissions WHERE id = ? AND user_id = ?`).bind(submissionID, userID),
  ]);
  return 1;
}

async function deleteOwnSubmissions(env: Env, userID: string): Promise<number> {
  const countRow = await env.DB.prepare(`SELECT COUNT(*) AS count FROM benchmark_submissions WHERE user_id = ?`)
    .bind(userID)
    .first<any>();
  await env.DB.batch([
    env.DB.prepare(`DELETE FROM benchmark_attempts WHERE submission_id IN (SELECT id FROM benchmark_submissions WHERE user_id = ?)`).bind(userID),
    env.DB.prepare(`DELETE FROM benchmark_question_results WHERE submission_id IN (SELECT id FROM benchmark_submissions WHERE user_id = ?)`).bind(userID),
    env.DB.prepare(`DELETE FROM benchmark_submissions WHERE user_id = ?`).bind(userID),
  ]);
  return int(countRow?.count);
}

async function getBearerUser(request: Request, env: Env): Promise<{ user: any; tokenID: string } | null> {
  const h = request.headers.get("authorization") || "";
  const m = /^Bearer\s+(.+)$/i.exec(h);
  if (!m) return null;
  const tokenHash = await hashSecret(m[1], env);
  const row = await env.DB.prepare(
    `SELECT access_tokens.id AS token_id, users.id, users.username, users.login, users.name, users.email,
            users.avatar_url, users.avatar_template, users.active, users.trust_level, users.silenced
     FROM access_tokens JOIN users ON users.id = access_tokens.user_id
     WHERE access_tokens.token_hash = ? AND access_tokens.revoked_at IS NULL
       AND (access_tokens.expires_at IS NULL OR access_tokens.expires_at > ?)`
  )
    .bind(tokenHash, iso(new Date()))
    .first<any>();
  if (!row) return null;
  await env.DB.prepare(`UPDATE access_tokens SET last_used_at = ? WHERE id = ?`).bind(iso(new Date()), row.token_id).run();
  return { tokenID: row.token_id, user: publicUser(row) };
}

async function getWebUser(request: Request, env: Env): Promise<any | null> {
  const token = parseCookies(request.headers.get("cookie")).ldgc_session;
  if (!token) return null;
  const row = await env.DB.prepare(
    `SELECT users.id, users.provider, users.provider_user_id, users.username, users.login, users.name, users.email,
            users.avatar_url, users.avatar_template, users.active, users.trust_level, users.silenced
     FROM web_sessions JOIN users ON users.id = web_sessions.user_id
     WHERE web_sessions.session_hash = ? AND web_sessions.revoked_at IS NULL AND web_sessions.expires_at > ?`
  )
    .bind(await hashSecret(token, env), iso(new Date()))
    .first<any>();
  return row || null;
}

function isAdminUser(user: any, env: Env): boolean {
  const linuxdoIDs = splitList(`${DEFAULT_ADMIN_LINUXDO_IDS},${env.ADMIN_LINUXDO_IDS || ""}`);
  const localIDs = splitList(env.ADMIN_USER_IDS || "");
  const providerUserID = String(user?.provider_user_id || "");
  const localUserID = String(user?.id || "");
  return linuxdoIDs.has(providerUserID) || localIDs.has(localUserID);
}

function splitList(value: string): Set<string> {
  const out = new Set<string>();
  for (const item of value.split(",")) {
    const trimmed = item.trim();
    if (trimmed) out.add(trimmed);
  }
  return out;
}

async function upsertLinuxDoUser(profile: any, env: Env): Promise<{ id: string; username: string }> {
  const providerUserID = str(profile.sub ?? profile.id ?? profile.user_id, MAX_STRING_LENGTH);
  if (!providerUserID) throw new Error("userinfo id missing");
  const username = str(profile.username ?? profile.login ?? profile.name ?? providerUserID, MAX_STRING_LENGTH);
  const login = str(profile.login ?? "", MAX_STRING_LENGTH);
  const name = str(profile.name ?? "", MAX_STRING_LENGTH);
  const email = str(profile.email ?? "", MAX_STRING_LENGTH);
  const avatarURL = str(profile.avatar_url ?? profile.avatar ?? "", MAX_STRING_LENGTH);
  const avatarTemplate = str(profile.avatar_template ?? "", MAX_STRING_LENGTH);
  const active = boolInt(profile.active);
  const trustLevel = optionalInt(profile.trust_level);
  const silenced = boolInt(profile.silenced);
  const profileJSON = safeLinuxDoProfileJSON(profile);
  const existing = await env.DB.prepare(`SELECT id, username FROM users WHERE provider = 'linuxdo' AND provider_user_id = ?`)
    .bind(providerUserID)
    .first<any>();
  if (existing) {
    await env.DB.prepare(
      `UPDATE users
       SET username = ?, login = ?, name = ?, email = ?, avatar_url = ?, avatar_template = ?,
           active = ?, trust_level = ?, silenced = ?, linuxdo_profile_json = ?, updated_at = ?
       WHERE id = ?`
    )
      .bind(username, login, name, email, avatarURL, avatarTemplate, active, trustLevel, silenced, profileJSON, iso(new Date()), existing.id)
      .run();
    return { id: existing.id, username };
  }
  const id = crypto.randomUUID();
  await env.DB.prepare(
    `INSERT INTO users
      (id, provider, provider_user_id, username, login, name, email, avatar_url, avatar_template,
       active, trust_level, silenced, linuxdo_profile_json, created_at, updated_at)
     VALUES (?, 'linuxdo', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
  )
    .bind(id, providerUserID, username, login, name, email, avatarURL, avatarTemplate, active, trustLevel, silenced, profileJSON, iso(new Date()), iso(new Date()))
    .run();
  return { id, username };
}

function publicUser(row: any): Record<string, unknown> {
  const username = row.username || row.login || "";
  return {
    id: row.id,
    username,
    login: row.login || "",
    name: row.name || "",
    email: row.email || "",
    avatar_url: row.avatar_url || "",
    avatar_template: row.avatar_template || "",
    linuxdo_url: linuxdoProfileURL(username),
    active: row.active == null ? null : !!row.active,
    trust_level: row.trust_level == null ? null : int(row.trust_level),
    silenced: row.silenced == null ? null : !!row.silenced,
  };
}

function submissionDisplayUser(user: any, anonymous: boolean): Record<string, unknown> {
  if (anonymous) {
    return {
      anonymous: true,
      display_name: "匿名",
      username: "",
      avatar_url: "",
      linuxdo_url: "",
    };
  }
  const username = str(user?.username || user?.login || "", MAX_STRING_LENGTH);
  return {
    anonymous: false,
    display_name: str(user?.name || username || "Linux.do 用户", MAX_STRING_LENGTH),
    username,
    avatar_url: str(user?.avatar_url || user?.avatar_template || "", MAX_STRING_LENGTH),
    linuxdo_url: linuxdoProfileURL(username),
  };
}

function linuxdoProfileURL(username: string): string {
  const clean = str(username, MAX_STRING_LENGTH).trim();
  return clean ? `https://linux.do/u/${encodeURIComponent(clean)}/summary` : "";
}

function safeLinuxDoProfileJSON(profile: any): string {
  const safe = {
    id: profile.id ?? null,
    sub: profile.sub ?? null,
    username: profile.username ?? null,
    login: profile.login ?? null,
    name: profile.name ?? null,
    email: profile.email ?? null,
    avatar_template: profile.avatar_template ?? null,
    avatar_url: profile.avatar_url ?? null,
    active: profile.active ?? null,
    trust_level: profile.trust_level ?? null,
    silenced: profile.silenced ?? null,
    external_ids: profile.external_ids ?? null,
  };
  return str(JSON.stringify(safe), MAX_METADATA_LENGTH);
}

function boolInt(value: unknown): number | null {
  return typeof value === "boolean" ? (value ? 1 : 0) : null;
}

function optionalInt(value: unknown): number | null {
  const n = Number(value);
  return Number.isInteger(n) ? n : null;
}

async function readJson<T>(request: Request): Promise<T> {
  enforceBodySize(request);
  try {
    const text = await request.text();
    if (text.length > MAX_JSON_BYTES) throw new APIError(413, "payload_too_large", "request body is too large");
    return JSON.parse(text) as T;
  } catch (err) {
    if (err instanceof APIError) throw err;
    throw new APIError(400, "bad_request", "invalid JSON body");
  }
}

function enforceBodySize(request: Request): void {
  const contentLength = Number(request.headers.get("content-length") || "0");
  if (contentLength > MAX_JSON_BYTES) {
    throw new APIError(413, "payload_too_large", "request body is too large");
  }
}

function json(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "content-type": "application/json; charset=utf-8" },
  });
}

function jsonError(message: string, status: number, code = "bad_request", requestID?: string): Response {
  return json({ error: message, code, request_id: requestID }, status);
}

function html(body: string, status = 200, scriptNonce = ""): Response {
  const headers = new Headers({ "content-type": "text/html; charset=utf-8" });
  if (scriptNonce) headers.set("x-ldgc-script-nonce", scriptNonce);
  return new Response(body, { status, headers });
}

function redirect(location: string, setCookie?: string | string[]): Response {
  const headers = new Headers({ location });
  for (const c of Array.isArray(setCookie) ? setCookie : setCookie ? [setCookie] : []) {
    headers.append("set-cookie", c);
  }
  return new Response(null, { status: 302, headers });
}

function withCommonHeaders(response: Response, request: Request, env: Env, requestID: string): Response {
  const headers = new Headers(response.headers);
  headers.set("x-request-id", requestID);
  headers.set("x-content-type-options", "nosniff");
  headers.set("referrer-policy", "same-origin");
  headers.set("permissions-policy", "camera=(), microphone=(), geolocation=()");
  headers.set("cross-origin-opener-policy", "same-origin");
  headers.set("cross-origin-resource-policy", "same-origin");

  const corsOrigin = allowedCORSOrigin(request, env);
  if (corsOrigin) {
    headers.set("access-control-allow-origin", corsOrigin);
    headers.set("vary", appendVary(headers.get("vary"), "Origin"));
    headers.set("access-control-allow-methods", "GET, POST, DELETE, OPTIONS");
    headers.set("access-control-allow-headers", "Authorization, Content-Type, Accept, Idempotency-Key");
    headers.set("access-control-max-age", "600");
  }

  if ((headers.get("content-type") || "").includes("text/html")) {
    const scriptNonce = headers.get("x-ldgc-script-nonce") || "";
    headers.delete("x-ldgc-script-nonce");
    const scriptSrc = scriptNonce
      ? `script-src 'nonce-${scriptNonce}' https://challenges.cloudflare.com`
      : "script-src https://challenges.cloudflare.com";
    headers.set(
      "content-security-policy",
      `default-src 'none'; ${scriptSrc}; style-src 'unsafe-inline'; img-src 'self' data: https://cdn.ldstatic.com; frame-src https://challenges.cloudflare.com; connect-src https://challenges.cloudflare.com; form-action 'self'; base-uri 'none'; frame-ancestors 'none'`
    );
  }
  return new Response(response.body, { status: response.status, statusText: response.statusText, headers });
}

function allowedCORSOrigin(request: Request, env: Env): string | null {
  const origin = request.headers.get("origin");
  if (!origin) return null;
  const allowed = new Set([baseOrigin(env)]);
  for (const value of (env.ALLOWED_ORIGINS || "").split(",")) {
    const trimmed = value.trim();
    if (trimmed) allowed.add(trimmed);
  }
  return allowed.has(origin) ? origin : null;
}

function appendVary(current: string | null, value: string): string {
  if (!current) return value;
  const parts = current.split(",").map((x) => x.trim().toLowerCase());
  return parts.includes(value.toLowerCase()) ? current : `${current}, ${value}`;
}

function matches(path: string, ...paths: string[]): boolean {
  return paths.includes(path);
}

function submissionItemPath(path: string): boolean {
  return /^\/api\/v1\/submissions\/[^/]+$/.test(path);
}

function knownPath(path: string): boolean {
  return submissionItemPath(path) || matches(
    path,
    "/",
    "/account",
    "/admin",
    "/admin/questions",
    "/admin/bridges",
    "/health",
    "/api/questions",
    "/api/v1/questions",
    "/api/v1/admin/questions",
    "/api/v1/admin/bridges",
    "/api/v1/admin/bridges/identify",
    "/api/v1/bridge-suggestions",
    "/api/device/start",
    "/api/v1/device-authorizations",
    "/api/device/poll",
    "/api/v1/device-authorizations/token",
    "/device",
    "/api/device/approve",
    "/api/v1/device-authorizations/approve",
    "/auth/linuxdo/start",
    "/auth/linuxdo/callback",
    "/api/me",
    "/api/v1/me",
    "/api/v1/submissions",
    "/account/submissions/delete",
    "/account/submissions/delete-all",
    "/account/bridge-suggestions",
    "/api/runs",
    "/api/v1/runs",
    "/api/logout",
    "/api/v1/sessions/logout",
    "/logout"
  );
}

function resultPage(title: string, message: string): string {
  return layoutPage(`${title} - LD-gpt-check`, `<section class="hero"><h1>${escapeHTML(title)}</h1><p>${escapeHTML(message)}</p><div class="actions"><a class="button" href="/account">返回账号页</a></div></section>`);
}

function linuxdoIcon(): string {
  return `<svg class="linuxdo-icon" viewBox="0 0 40 40" role="img" aria-label="Linux.do" focusable="false">
    <rect width="40" height="40" rx="10" fill="#1d4ed8"></rect>
    <path d="M10 12.5h20v5.5H18.1v3.3h9.9v5.1h-9.9V32H10V12.5Z" fill="#fff"></path>
    <circle cx="30" cy="30" r="3" fill="#67e8f9"></circle>
  </svg>`;
}

function userIdentityBlock(user: any): string {
  const username = str(user.username || user.login || "", MAX_STRING_LENGTH);
  const display = str(user.name || username || user.id, MAX_STRING_LENGTH);
  const avatar = str(user.avatar_url || user.avatar_template || "", MAX_STRING_LENGTH);
  const profile = linuxdoProfileURL(username);
  const avatarHTML = avatar
    ? `<img class="user-avatar" src="${escapeHTML(avatar)}" alt="${escapeHTML(display)}">`
    : `<span class="user-avatar user-avatar-fallback">${escapeHTML(display.slice(0, 1).toUpperCase() || "U")}</span>`;
  const nameHTML = profile
    ? `<a class="user-name" href="${escapeHTML(profile)}" target="_blank" rel="noreferrer">${escapeHTML(display)}</a>`
    : `<span class="user-name">${escapeHTML(display)}</span>`;
  return `<div class="user-head">${profile ? `<a href="${escapeHTML(profile)}" target="_blank" rel="noreferrer">${avatarHTML}</a>` : avatarHTML}<div><h1>${nameHTML}</h1><p class="user-meta">${username ? "@" + escapeHTML(username) : escapeHTML(user.id)}</p></div></div>`;
}

function layoutPage(title: string, content: string): string {
  return `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>${escapeHTML(title)}</title>
<style>
:root{color-scheme:light;--text:#0f172a;--muted:#64748b;--line:#dbeafe;--brand:#2563eb;--brand2:#06b6d4;--bg:#f7fbff}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:var(--text);background:linear-gradient(135deg,rgba(37,99,235,.1),transparent 34%),linear-gradient(225deg,rgba(6,182,212,.14),transparent 38%),var(--bg);padding:24px;line-height:1.5}
main{width:min(100%,920px);margin:0 auto}.nav{display:flex;align-items:center;justify-content:space-between;margin-bottom:18px;border:1px solid rgba(191,219,254,.9);background:rgba(255,255,255,.86);border-radius:14px;padding:12px 14px;box-shadow:0 16px 50px rgba(37,99,235,.12)}.brand{font-weight:800;color:#1d4ed8;text-decoration:none}.nav a{color:#334155;text-decoration:none;font-size:14px}
.hero,.panel,.grid article{border:1px solid rgba(191,219,254,.9);background:rgba(255,255,255,.88);box-shadow:0 20px 70px rgba(37,99,235,.13);backdrop-filter:blur(18px);border-radius:16px;padding:24px}.hero{margin-bottom:16px}.badge{display:inline-flex;margin-bottom:12px;border:1px solid #bfdbfe;border-radius:999px;padding:5px 10px;font:12px ui-monospace,SFMono-Regular,Menlo,monospace;color:#1d4ed8;background:#eff6ff}h1{margin:0;font-size:34px;line-height:1.12;letter-spacing:0}h2{margin:0 0 8px;font-size:20px}p{margin:10px 0 0;color:var(--muted)}.user-head{display:flex;align-items:center;gap:14px}.user-avatar{display:block;width:64px;height:64px;border-radius:16px;object-fit:cover;border:1px solid #bfdbfe;background:#eff6ff;box-shadow:0 12px 30px rgba(37,99,235,.14)}.user-avatar-fallback{display:grid;place-items:center;font-size:24px;font-weight:800;color:#1d4ed8}.user-name{color:var(--text);text-decoration:none}.user-name:hover{color:#1d4ed8}.user-meta{margin-top:4px;font:13px ui-monospace,SFMono-Regular,Menlo,monospace;color:#64748b}.inline-link{color:#1d4ed8;text-decoration:none}.inline-link:hover{text-decoration:underline}.admin-list{display:grid;gap:10px;margin-top:12px}.admin-list a{display:block;border:1px solid #dbeafe;border-radius:10px;background:#f8fafc;padding:14px;text-decoration:none;color:var(--text)}.admin-list a:hover{border-color:#93c5fd;background:#eff6ff}.admin-list strong{display:block}.admin-list span{display:block;margin-top:4px;color:#64748b;font-size:13px}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:20px;align-items:center}.actions form{margin:0}.panel input{min-height:42px;border:1px solid #cbd5e1;border-radius:10px;padding:9px 12px;font:600 14px system-ui,sans-serif}button,a.button{appearance:none;border:1px solid var(--brand);border-radius:10px;background:linear-gradient(135deg,var(--brand),var(--brand2));color:#fff;min-height:42px;padding:10px 14px;font:700 14px system-ui,sans-serif;text-decoration:none;display:inline-flex;align-items:center;justify-content:center;cursor:pointer}.secondary{border-color:#cbd5e1;background:#fff;color:#334155}.danger{border-color:#dc2626;background:#dc2626;color:#fff}.small{min-height:32px;padding:6px 10px;font-size:12px}.delete-all{margin-top:16px;border-top:1px solid #e2e8f0;padding-top:16px}.delete-all label{display:block;color:#64748b;font-size:14px}.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-bottom:16px}.grid span{display:block;font:12px ui-monospace,SFMono-Regular,Menlo,monospace;color:#64748b}.grid strong{display:block;margin-top:6px;overflow-wrap:anywhere}.panel{margin-bottom:16px}pre{overflow:auto;border:1px solid #dbeafe;background:#f8fafc;border-radius:10px;padding:12px;color:#1d4ed8}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;font-size:14px}th,td{text-align:left;border-bottom:1px solid #e2e8f0;padding:10px 8px;white-space:nowrap}th{color:#64748b;font-weight:650}td form{margin:0}
.login-actions{margin-top:22px}.linuxdo-button{display:inline-flex;align-items:center;gap:12px;min-height:58px;border:1px solid #1d4ed8;border-radius:14px;background:linear-gradient(135deg,#1d4ed8,#06b6d4);box-shadow:0 18px 45px rgba(37,99,235,.24);color:#fff;padding:10px 16px;text-decoration:none;transition:transform .16s ease,box-shadow .16s ease,filter .16s ease}.linuxdo-button:hover{transform:translateY(-1px);box-shadow:0 22px 56px rgba(37,99,235,.3);filter:saturate(1.08)}.linuxdo-button span{display:grid;gap:2px;text-align:left}.linuxdo-button strong{color:#fff;font-size:15px;line-height:1.2}.linuxdo-button small{color:rgba(255,255,255,.78);font-size:12px;line-height:1.3}.linuxdo-icon{width:38px;height:38px;flex:0 0 auto;border-radius:10px;box-shadow:inset 0 0 0 1px rgba(255,255,255,.22)}
@media(max-width:680px){body{padding:12px}.grid{grid-template-columns:1fr}h1{font-size:28px}button,a.button,.linuxdo-button{width:100%}.actions form{width:100%}.linuxdo-button{justify-content:flex-start}}
</style></head>
<body><main><nav class="nav"><a class="brand" href="/account">LD-gpt-check</a><a href="/device">授权 CLI</a></nav>${content}</main></body></html>`;
}

function cookie(name: string, value: string, env: Env, maxAge: number): string {
  const secure = env.BASE_URL.startsWith("https://") ? "; Secure" : "";
  return `${name}=${encodeURIComponent(value)}; Path=/; HttpOnly; SameSite=Lax; Max-Age=${maxAge}${secure}`;
}

function parseCookies(header: string | null): Record<string, string> {
  const out: Record<string, string> = {};
  for (const part of (header || "").split(";")) {
    const [k, ...rest] = part.trim().split("=");
    if (!k) continue;
    try {
      out[k] = decodeURIComponent(rest.join("="));
    } catch {
      out[k] = "";
    }
  }
  return out;
}

async function hashSecret(value: string, env: Env): Promise<string> {
  const pepper = env.TOKEN_SECRET || env.SESSION_SECRET || "";
  if (!pepper) throw new APIError(500, "server_misconfigured", "server is not configured");
  const data = new TextEncoder().encode(`${pepper}:${value}`);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

async function randomToken(prefix: string): Promise<string> {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  const raw = btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
  return `${prefix}_${raw}`;
}

function cspNonce(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function numericCode(): string {
  const n = crypto.getRandomValues(new Uint32Array(3));
  const parts = [...n].map((x) => String(x % 1000).padStart(3, "0"));
  return parts.join("-");
}

function normalizeCode(code: string): string {
  return code.replace(/\D/g, "");
}

function iso(d: Date): string {
  return d.toISOString();
}

function addSeconds(d: Date, seconds: number): Date {
  return new Date(d.getTime() + seconds * 1000);
}

function secondsBetween(thenISO: string, now: Date): number {
  return (now.getTime() - new Date(thenISO).getTime()) / 1000;
}

function safeRedirect(path: string): string {
  return path.startsWith("/") && !path.startsWith("//") ? path : "/device";
}

function baseOrigin(env: Env): string {
  return new URL(env.BASE_URL.replace(/\/$/, "")).origin;
}

function enforceSameOrigin(request: Request, env: Env): void {
  const expected = baseOrigin(env);
  const origin = request.headers.get("origin");
  if (origin && origin !== expected) throw new APIError(403, "forbidden", "cross-origin request denied");
  const referer = request.headers.get("referer");
  if (!origin && referer && safeOrigin(referer) !== expected) {
    throw new APIError(403, "forbidden", "cross-origin request denied");
  }
}

function safeOrigin(value: string): string {
  try {
    return new URL(value).origin;
  } catch {
    return "";
  }
}

async function verifyTurnstileIfConfigured(request: Request, env: Env, input: Record<string, any>): Promise<void> {
  if (!env.TURNSTILE_SECRET_KEY || !env.TURNSTILE_SITE_KEY) return;
  const token = String(input["cf-turnstile-response"] || input.turnstile_token || "");
  if (!token) throw new APIError(400, "turnstile_required", "human verification is required");

  const resp = await fetch("https://challenges.cloudflare.com/turnstile/v0/siteverify", {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      secret: env.TURNSTILE_SECRET_KEY,
      response: token,
      remoteip: clientIP(request),
    }),
  });
  if (!resp.ok) throw new APIError(502, "upstream_error", "human verification failed");
  const result: any = await resp.json();
  if (!result.success) throw new APIError(400, "turnstile_failed", "human verification failed");
}

async function enforceRateLimit(request: Request, env: Env, action: string, limit: number, windowSeconds: number): Promise<void> {
  await rateLimit(env, `${action}:ip:${clientIP(request)}`, limit, windowSeconds);
}

async function enforceUserRateLimit(env: Env, userID: string, action: string, limit: number, windowSeconds: number): Promise<void> {
  await rateLimit(env, `${action}:user:${userID}`, limit, windowSeconds);
}

async function rateLimit(env: Env, key: string, limit: number, windowSeconds: number): Promise<void> {
  const now = new Date();
  const row = await env.DB.prepare(`SELECT window_start, count FROM rate_limits WHERE key = ?`).bind(key).first<any>();
  if (!row || secondsBetween(row.window_start, now) >= windowSeconds) {
    await env.DB.prepare(
      `INSERT INTO rate_limits (key, window_start, count)
       VALUES (?, ?, 1)
       ON CONFLICT(key) DO UPDATE SET window_start = excluded.window_start, count = 1`
    )
      .bind(key, iso(now))
      .run();
    return;
  }
  if (Number(row.count) >= limit) throw new APIError(429, "rate_limited", "too many requests");
  await env.DB.prepare(`UPDATE rate_limits SET count = count + 1 WHERE key = ?`).bind(key).run();
}

function clientIP(request: Request): string {
  return request.headers.get("cf-connecting-ip") || request.headers.get("x-forwarded-for")?.split(",")[0]?.trim() || "unknown";
}

function escapeHTML(s: string): string {
  return s.replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]!));
}

function formatPercent(v: number): string {
  if (!Number.isFinite(v)) return "-";
  return `${v.toFixed(1)}%`;
}

function channelLabel(channel: unknown, bridgeName: unknown): string {
  if (channel === "official") return "官方";
  if (channel === "bridge") return escapeHTML(str(bridgeName || "中转站", MAX_STRING_LENGTH));
  if (channel === "unknown_bridge") return "未知中转站";
  return "-";
}

function formatDate(v: unknown): string {
  if (typeof v !== "string" || !v) return "-";
  const d = new Date(v);
  if (!Number.isFinite(d.getTime())) return v;
  return d.toISOString().replace("T", " ").replace(/\.\d{3}Z$/, " UTC");
}

function validateQuestionBank(bank: any): string | null {
  if (!bank || typeof bank !== "object" || Array.isArray(bank)) return "题库必须是 JSON object。";
  if (bank.schema_version != null && String(bank.schema_version) !== "1") return "schema_version 目前只支持 1。";
  if (!Array.isArray(bank.questions)) return "questions 必须是数组。";
  if (bank.questions.length < 1 || bank.questions.length > MAX_QUESTIONS) return `questions 数量必须在 1 到 ${MAX_QUESTIONS} 之间。`;
  const seen = new Set<string>();
  for (const q of bank.questions) {
    if (!q || typeof q !== "object" || Array.isArray(q)) return "每道题必须是 object。";
    if (!requiredString(q.id)) return "每道题必须有短 id。";
    if (seen.has(q.id)) return `题目 id 重复：${q.id}`;
    seen.add(q.id);
    if (!requiredString(q.version)) return `题目 ${q.id} 必须有 version。`;
    if (!requiredString(q.title)) return `题目 ${q.id} 必须有 title。`;
    if (typeof q.prompt !== "string" || q.prompt.trim() === "" || q.prompt.length > 12000) return `题目 ${q.id} prompt 必须是非空字符串且不能过长。`;
    if (q.tags != null && !stringArray(q.tags, 20, 64)) return `题目 ${q.id} tags 必须是短字符串数组。`;
    const grader = q.grader;
    if (!grader || typeof grader !== "object" || Array.isArray(grader)) return `题目 ${q.id} 必须有 grader。`;
    const typ = String(grader.type || "");
    if (!["number", "exact", "regex"].includes(typ)) return `题目 ${q.id} grader.type 只能是 number、exact 或 regex。`;
    if ((typ === "number" || typ === "exact") && typeof grader.expected !== "string") return `题目 ${q.id} grader.expected 必须是字符串。`;
    if (typ === "number" && grader.expected.trim() === "") return `题目 ${q.id} number expected 不能为空。`;
    if (typ === "number" && !Number.isFinite(Number(grader.expected))) return `题目 ${q.id} number expected 必须是数字。`;
    if (typ === "exact" && grader.expected === "") return `题目 ${q.id} exact expected 不能为空。`;
    if (typ === "regex") {
      if (typeof grader.pattern !== "string" || grader.pattern === "") return `题目 ${q.id} regex pattern 不能为空。`;
      try {
        new RegExp(grader.pattern);
      } catch {
        return `题目 ${q.id} regex pattern 无效。`;
      }
    }
    for (const key of ["independent_match", "case_sensitive", "trim_space"]) {
      if (grader[key] != null && typeof grader[key] !== "boolean") return `题目 ${q.id} grader.${key} 必须是 boolean。`;
    }
    if (grader.tolerance != null && !validNumber(Number(grader.tolerance), 0, Number.MAX_SAFE_INTEGER)) return `题目 ${q.id} grader.tolerance 必须是非负数字。`;
  }
  return null;
}

function validateSubmissionPayload(p: any): string | null {
  if (!p || typeof p !== "object" || Array.isArray(p)) return "request body must be a JSON object";
  if (!requiredString(p.upload_id)) return "upload_id is required";
  if (!requiredString(p.client_version)) return "client_version is required";
  if (!requiredString(p.model)) return "model is required";
  if (!requiredString(p.reasoning_effort)) return "reasoning_effort is required";
  if (!validInt(p.upload_schema_version, 4, 10)) return "upload_schema_version must be an integer between 4 and 10";
  if (!validInt(p.question_count, 1, MAX_QUESTIONS)) return `question_count must be an integer between 1 and ${MAX_QUESTIONS}`;
  if (!validInt(p.attempt_count, 1, MAX_ATTEMPTS)) return `attempt_count must be an integer between 1 and ${MAX_ATTEMPTS}`;
  if (!validInt(p.correct, 0, int(p.attempt_count))) return "correct must be an integer between 0 and attempt_count";
  if (!validNumber(p.accuracy, 0, 100)) return "accuracy must be a number between 0 and 100";
  for (const key of ["avg_input_tokens", "avg_output_tokens", "avg_reason_tokens", "avg_time_seconds", "avg_tps"]) {
    if (!validNumber(p[key], 0, Number.MAX_SAFE_INTEGER)) return `${key} must be a non-negative number`;
  }
  if (p.anonymous != null && typeof p.anonymous !== "boolean") return "anonymous must be a boolean";
  for (const key of ["started_at", "finished_at", "question_suite", "client_timezone"]) {
    if (p[key] != null && (typeof p[key] !== "string" || p[key].length > MAX_STRING_LENGTH)) return `${key} must be a short string`;
  }
  if (p.duration_seconds != null && !validNumber(p.duration_seconds, 0, Number.MAX_SAFE_INTEGER)) return "duration_seconds must be a non-negative number";
  if (!requiredString(p.os)) return "os is required";
  if (!requiredString(p.arch)) return "arch is required";
  if (!requiredString(p.codex_version)) return "codex_version is required";
  if (p.codex_model_source != null && !["explicit", "codex_config", "unknown"].includes(String(p.codex_model_source))) return "codex_model_source is invalid";
  if (!normalizeProviderBaseURL(p.codex_provider_base_url)) return "codex_provider_base_url must be a valid https URL";
  for (const key of ["codex_model_provider", "codex_provider_host", "codex_sandbox"]) {
    if (p[key] != null && (typeof p[key] !== "string" || p[key].length > MAX_STRING_LENGTH)) return `${key} must be a short string`;
  }
  for (const key of ["codex_ephemeral", "codex_skip_git_repo_check"]) {
    if (p[key] != null && typeof p[key] !== "boolean") return `${key} must be a boolean`;
  }
  if (p.codex_disabled_features != null && !stringArray(p.codex_disabled_features, 20, 64)) return "codex_disabled_features must be a short string array";
  if (p.codex_invocation != null && (typeof p.codex_invocation !== "string" || p.codex_invocation.length > MAX_METADATA_LENGTH)) return "codex_invocation must be a short string";
  if (!Array.isArray(p.questions)) return "questions must be an array";
  if (p.questions.length !== int(p.question_count)) return "questions length must equal question_count";
  for (const q of p.questions) {
    if (!q || typeof q !== "object" || Array.isArray(q)) return "each question must be an object";
    if (!requiredString(q.question_id)) return "question_id is required";
    if (!requiredString(q.question_version)) return "question_version is required";
    if (!requiredString(q.question_title)) return "question_title is required";
    if (!["number", "exact", "regex"].includes(String(q.grader_type))) return "grader_type must be number, exact, or regex";
    if (!requiredString(q.expected_answer)) return "expected_answer is required";
    if (!requiredString(q.prompt_hash)) return "prompt_hash is required";
    if (!validInt(q.tests, 1, MAX_ATTEMPTS)) return "question tests must be a positive integer";
    if (!validInt(q.correct, 0, int(q.tests))) return "question correct must be between 0 and tests";
    if (!validNumber(q.accuracy, 0, 100)) return "question accuracy must be a number between 0 and 100";
    for (const key of ["avg_input_tokens", "avg_output_tokens", "avg_reason_tokens", "avg_time_seconds", "avg_tps"]) {
      if (!validNumber(q[key], 0, Number.MAX_SAFE_INTEGER)) return `question ${key} must be a non-negative number`;
    }
  }
  if (!Array.isArray(p.attempts)) return "attempts must be an array";
  if (p.attempts.length !== int(p.attempt_count)) return "attempts length must equal attempt_count";
  for (const a of p.attempts) {
    if (!a || typeof a !== "object" || Array.isArray(a)) return "each attempt must be an object";
    if (!requiredString(a.question_id)) return "attempt question_id is required";
    if (!requiredString(a.question_version)) return "attempt question_version is required";
    if (!validInt(a.case_index, 1, MAX_ATTEMPTS)) return "case_index must be a positive integer";
    if (!["completed", "failed"].includes(String(a.status || "completed"))) return "attempt status must be completed or failed";
    if (typeof a.is_correct !== "boolean") return "attempt is_correct must be a boolean";
    if (typeof a.expected_answer !== "string") return "attempt expected_answer must be a string";
    if (typeof a.extracted_answer !== "string") return "attempt extracted_answer must be a string";
    if (typeof a.failure_reason !== "undefined" && typeof a.failure_reason !== "string") return "attempt failure_reason must be a string";
    if (a.failure_reason && !["no_answer", "wrong_answer", "parse_error", "tool_used", "codex_failed", "timeout", "unknown"].includes(String(a.failure_reason))) {
      return "attempt failure_reason is invalid";
    }
    if (typeof a.answer_preview !== "string") return "attempt answer_preview must be a string";
    if ("full_answer" in a || "prompt" in a || "prompt_text" in a) return "attempt must not include full answer or prompt";
    if (a.answer_preview_truncated != null && typeof a.answer_preview_truncated !== "boolean") return "answer_preview_truncated must be a boolean";
    if (a.answer_hash != null && (typeof a.answer_hash !== "string" || a.answer_hash.length > 64)) return "answer_hash must be a sha256 hex string";
    for (const key of ["input_tokens", "output_tokens", "reasoning_tokens"]) {
      if (!validInt(a[key], 0, Number.MAX_SAFE_INTEGER)) return `attempt ${key} must be a non-negative integer`;
    }
    for (const key of ["cached_input_tokens", "total_tokens", "event_count", "answer_chars"]) {
      if (a[key] != null && !validInt(a[key], 0, Number.MAX_SAFE_INTEGER)) return `attempt ${key} must be a non-negative integer`;
    }
    if (a.codex_thread_id != null && (typeof a.codex_thread_id !== "string" || a.codex_thread_id.length > MAX_STRING_LENGTH)) return "codex_thread_id must be a short string";
    if (a.event_types != null && !stringArray(a.event_types, 100, 128)) return "event_types must be a short string array";
    if (a.tool_event_detected != null && typeof a.tool_event_detected !== "boolean") return "tool_event_detected must be a boolean";
    for (const key of ["time_seconds", "tps"]) {
      if (!validNumber(a[key], 0, Number.MAX_SAFE_INTEGER)) return `attempt ${key} must be a non-negative number`;
    }
    for (const key of ["error_code", "started_at", "finished_at"]) {
      if (a[key] != null && (typeof a[key] !== "string" || a[key].length > MAX_STRING_LENGTH)) return `attempt ${key} must be a short string`;
    }
    if (a.timeout_seconds != null && !validNumber(a.timeout_seconds, 0, Number.MAX_SAFE_INTEGER)) return "attempt timeout_seconds must be a non-negative number";
  }
  return null;
}

async function upsertUnknownProviderSuggestion(env: Env, userID: string, provider: { baseURL: string; host: string }): Promise<void> {
  try {
    await upsertBridgeSuggestion(env, {
      userID,
      baseURL: provider.baseURL,
      host: provider.host,
      source: "upload",
      submittedName: "",
    });
  } catch (err) {
    console.warn("failed to record bridge suggestion", { error: err instanceof Error ? err.message : String(err) });
  }
}

async function upsertBridgeSuggestion(
  env: Env,
  input: { userID: string; baseURL: string; host: string; source: string; submittedName: string }
): Promise<Record<string, unknown>> {
  const now = iso(new Date());
  const existing = await env.DB.prepare(`SELECT id, occurrence_count, status FROM bridge_suggestions WHERE base_url = ?`)
    .bind(input.baseURL)
    .first<any>();
  if (existing?.id) {
    await env.DB.prepare(
      `UPDATE bridge_suggestions
       SET user_id = COALESCE(user_id, ?),
           source = CASE WHEN source = 'upload' AND ? = 'user' THEN 'user' ELSE source END,
           submitted_name = CASE WHEN ? != '' THEN ? ELSE submitted_name END,
           status = CASE WHEN status = 'rejected' THEN 'pending' ELSE status END,
           occurrence_count = occurrence_count + 1,
           last_seen_at = ?,
           updated_at = ?
       WHERE id = ?`
    )
      .bind(input.userID, input.source, input.submittedName, input.submittedName, now, now, existing.id)
      .run();
    return { id: existing.id, base_url: input.baseURL, status: existing.status || "pending" };
  }
  const detected = await identifyBridgeCandidate(env, input.baseURL, input.submittedName);
  const id = crypto.randomUUID();
  await env.DB.prepare(
    `INSERT INTO bridge_suggestions
       (id, user_id, base_url, host, source, submitted_name, page_title, icon_url,
        ai_name, ai_slug, ai_confidence, ai_reason, status, occurrence_count, created_at, updated_at, last_seen_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', 1, ?, ?, ?)`
  )
    .bind(
      id,
      input.userID,
      input.baseURL,
      input.host,
      input.source,
      input.submittedName,
      detected.page_title,
      detected.icon_url,
      detected.detected_name,
      detected.slug,
      detected.confidence,
      detected.reason,
      now,
      now,
      now
    )
    .run();
  return { id, base_url: input.baseURL, status: "pending", detected_name: detected.detected_name, icon_url: detected.icon_url };
}

async function identifyBridgeCandidate(env: Env, rawBaseURL: unknown, submittedName = ""): Promise<Record<string, unknown>> {
  const baseURL = normalizeProviderBaseURL(rawBaseURL);
  if (!baseURL) throw new APIError(400, "bad_request", "base_url must be a valid https URL");
  const host = hostFromProviderBaseURL(baseURL);
  const homepageURL = providerHomepageURL(baseURL);
  const page = await fetchBridgePageInfo(homepageURL);
  const fallbackName = str(submittedName || page.title || host.replace(/^api\./, ""), MAX_STRING_LENGTH);
  const ai = await identifyBridgeWithAI(env, {
    baseURL,
    host,
    homepageURL,
    submittedName,
    pageTitle: page.title,
  });
  const detectedName = str(ai?.name || fallbackName, MAX_STRING_LENGTH);
  const slug = slugify(str(ai?.slug || detectedName, MAX_STRING_LENGTH)) || slugify(host) || `bridge-${crypto.randomUUID().slice(0, 8)}`;
  const confidence = typeof ai?.confidence === "number" && Number.isFinite(ai.confidence) ? Math.max(0, Math.min(1, ai.confidence)) : page.title ? 0.55 : 0.35;
  const reason = str(ai?.reason || (page.title ? "matched from page title and host" : "matched from host"), MAX_METADATA_LENGTH);
  return {
    base_url: baseURL,
    host,
    homepage_url: homepageURL,
    page_title: page.title,
    icon_url: page.iconURL,
    detected_name: detectedName,
    slug,
    confidence,
    reason,
    ai_used: !!ai,
    ai_configured: !!(env.BRIDGE_AI_BASE_URL && env.BRIDGE_AI_API_KEY),
  };
}

async function fetchBridgePageInfo(homepageURL: string): Promise<{ title: string; iconURL: string }> {
  let title = "";
  let iconURL = "";
  try {
    const resp = await fetch(homepageURL, {
      headers: { accept: "text/html,application/xhtml+xml" },
      redirect: "follow",
      signal: AbortSignal.timeout(6000),
    });
    const finalURL = resp.url || homepageURL;
    if (resp.ok) {
      const text = await responseTextLimit(resp, 64 * 1024);
      title = str(decodeHTMLEntities(extractTitle(text)), MAX_STRING_LENGTH);
      const iconCandidate = extractIconURL(text, finalURL);
      iconURL = (await firstReachableIconURL(iconCandidate, finalURL)) || "";
    }
    if (!iconURL) iconURL = (await firstReachableIconURL("", finalURL)) || "";
  } catch {
    iconURL = "";
  }
  return { title, iconURL };
}

async function responseTextLimit(response: Response, maxBytes: number): Promise<string> {
  if (!response.body) return "";
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (total < maxBytes) {
    const { value, done } = await reader.read();
    if (done || !value) break;
    const slice = value.byteLength > maxBytes - total ? value.slice(0, maxBytes - total) : value;
    chunks.push(slice);
    total += slice.byteLength;
  }
  try {
    await reader.cancel();
  } catch {
    // Ignore cancellation failures; the bounded read already has the data needed.
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder("utf-8", { fatal: false }).decode(bytes);
}

function extractTitle(htmlText: string): string {
  return /<title[^>]*>([\s\S]*?)<\/title>/i.exec(htmlText)?.[1]?.replace(/\s+/g, " ").trim() || "";
}

function extractIconURL(htmlText: string, baseURL: string): string {
  const links = htmlText.match(/<link\b[^>]*>/gi) || [];
  for (const link of links) {
    const rel = attrValue(link, "rel").toLowerCase();
    if (!rel.split(/\s+/).some((part) => ["icon", "shortcut", "apple-touch-icon", "mask-icon"].includes(part))) continue;
    const href = attrValue(link, "href");
    const resolved = resolveHTTPSURL(href, baseURL);
    if (resolved) return resolved;
  }
  return "";
}

function attrValue(tag: string, name: string): string {
  const re = new RegExp(`${name}\\s*=\\s*("([^"]*)"|'([^']*)'|([^\\s>]+))`, "i");
  const match = re.exec(tag);
  return match ? str(match[2] || match[3] || match[4] || "", MAX_URL_LENGTH) : "";
}

async function firstReachableIconURL(iconCandidate: string, pageURL: string): Promise<string> {
  const candidates = [iconCandidate, resolveHTTPSURL("/favicon.ico", pageURL)].filter(Boolean);
  for (const candidate of candidates) {
    if (await iconURLReachable(candidate)) return candidate;
  }
  return "";
}

async function iconURLReachable(url: string): Promise<boolean> {
  if (!normalizePublicHTTPSURL(url)) return false;
  for (const method of ["HEAD", "GET"]) {
    try {
      const resp = await fetch(url, { method, redirect: "follow", signal: AbortSignal.timeout(5000) });
      if (resp.ok) {
        const contentType = resp.headers.get("content-type") || "";
        return !contentType || /^image\//i.test(contentType) || /icon/i.test(contentType);
      }
    } catch {
      // Try the next method or candidate.
    }
  }
  return false;
}

async function identifyBridgeWithAI(
  env: Env,
  input: { baseURL: string; host: string; homepageURL: string; submittedName: string; pageTitle: string }
): Promise<{ name?: string; slug?: string; confidence?: number; reason?: string } | null> {
  if (!env.BRIDGE_AI_BASE_URL || !env.BRIDGE_AI_API_KEY) return null;
  const endpoint = `${env.BRIDGE_AI_BASE_URL.replace(/\/+$/, "")}/chat/completions`;
  try {
    const resp = await fetch(endpoint, {
      method: "POST",
      headers: {
        authorization: `Bearer ${env.BRIDGE_AI_API_KEY}`,
        "content-type": "application/json",
      },
      body: JSON.stringify({
        model: env.BRIDGE_AI_MODEL || "gpt-4.1-mini",
        temperature: 0,
        max_tokens: 180,
        response_format: { type: "json_object" },
        messages: [
          { role: "system", content: "Return compact JSON only. Identify the likely AI API bridge/provider name from URL and page metadata." },
          {
            role: "user",
            content: JSON.stringify({
              base_url: input.baseURL,
              host: input.host,
              homepage_url: input.homepageURL,
              submitted_name: input.submittedName,
              page_title: input.pageTitle,
              expected_schema: { name: "string", slug: "kebab-case string", confidence: "0..1 number", reason: "short string" },
            }),
          },
        ],
      }),
      signal: AbortSignal.timeout(10000),
    });
    if (!resp.ok) return null;
    const data: any = await resp.json();
    const content = String(data?.choices?.[0]?.message?.content || "");
    const parsed = parseJSONFromText(content);
    if (!parsed || typeof parsed !== "object") return null;
    return {
      name: str((parsed as any).name, MAX_STRING_LENGTH),
      slug: slugify(str((parsed as any).slug, MAX_STRING_LENGTH)),
      confidence: Number((parsed as any).confidence),
      reason: str((parsed as any).reason, MAX_METADATA_LENGTH),
    };
  } catch {
    return null;
  }
}

function parseJSONFromText(text: string): any | null {
  try {
    return JSON.parse(text);
  } catch {
    const match = /\{[\s\S]*\}/.exec(text);
    if (!match) return null;
    try {
      return JSON.parse(match[0]);
    } catch {
      return null;
    }
  }
}

async function classifyProviderBaseURL(env: Env, raw: string): Promise<{ baseURL: string; host: string; channel: string; bridgeID: string | null; bridgeName: string }> {
  const baseURL = normalizeProviderBaseURL(raw);
  const host = hostFromProviderBaseURL(baseURL);
  if (officialProviderBaseURL(baseURL)) {
    return { baseURL, host, channel: "official", bridgeID: null, bridgeName: "" };
  }
  const row = await env.DB.prepare(
    `SELECT bridge_base_urls.bridge_id, bridges.name
     FROM bridge_base_urls
     JOIN bridges ON bridges.id = bridge_base_urls.bridge_id
     WHERE bridge_base_urls.base_url = ?
       AND bridge_base_urls.is_active = 1
       AND bridges.is_active = 1
     LIMIT 1`
  )
    .bind(baseURL)
    .first<any>();
  if (row?.bridge_id) {
    return { baseURL, host, channel: "bridge", bridgeID: str(row.bridge_id, MAX_STRING_LENGTH), bridgeName: str(row.name, MAX_STRING_LENGTH) };
  }
  const hostRows = await env.DB.prepare(
    `SELECT bridge_base_urls.bridge_id, bridges.name
     FROM bridge_base_urls
     JOIN bridges ON bridges.id = bridge_base_urls.bridge_id
     WHERE bridge_base_urls.host = ?
       AND bridge_base_urls.is_active = 1
       AND bridges.is_active = 1
     GROUP BY bridge_base_urls.bridge_id, bridges.name
     LIMIT 2`
  )
    .bind(host)
    .all<any>();
  const matches = hostRows.results ?? [];
  if (matches.length === 1 && matches[0]?.bridge_id) {
    return { baseURL, host, channel: "bridge", bridgeID: str(matches[0].bridge_id, MAX_STRING_LENGTH), bridgeName: str(matches[0].name, MAX_STRING_LENGTH) };
  }
  return { baseURL, host, channel: "unknown_bridge", bridgeID: null, bridgeName: "" };
}

function normalizeBridgeBaseURLs(value: unknown): Array<{ baseURL: string; host: string }> {
  const rawItems = Array.isArray(value) ? value : typeof value === "string" ? value.split(/\n|,/) : [];
  const seen = new Set<string>();
  const result: Array<{ baseURL: string; host: string }> = [];
  for (const item of rawItems) {
    const raw = typeof item === "string" ? item : item && typeof item === "object" ? String((item as any).base_url || "") : "";
    const baseURL = normalizeProviderBaseURL(raw);
    if (!baseURL || seen.has(baseURL)) continue;
    seen.add(baseURL);
    result.push({ baseURL, host: hostFromProviderBaseURL(baseURL) });
  }
  return result;
}

function normalizeProviderBaseURL(raw: unknown): string {
  if (typeof raw !== "string") return "";
  const trimmed = raw.trim();
  if (!trimmed || trimmed.length > MAX_BASE_URL_LENGTH) return "";
  try {
    const url = new URL(trimmed);
    if (url.protocol !== "https:" || !url.hostname) return "";
    url.protocol = "https:";
    url.hostname = url.hostname.toLowerCase();
    url.username = "";
    url.password = "";
    url.search = "";
    url.hash = "";
    url.pathname = url.pathname.replace(/\/+$/, "");
    return url.toString().replace(/\/$/, "");
  } catch {
    return "";
  }
}

function normalizePublicHTTPSURL(raw: unknown): string {
  if (typeof raw !== "string") return "";
  const trimmed = raw.trim();
  if (!trimmed || trimmed.length > MAX_URL_LENGTH) return "";
  try {
    const url = new URL(trimmed);
    if (url.protocol !== "https:" || !url.hostname || privateHostname(url.hostname)) return "";
    url.username = "";
    url.password = "";
    url.hash = "";
    return url.toString();
  } catch {
    return "";
  }
}

function providerHomepageURL(baseURL: string): string {
  try {
    const url = new URL(baseURL);
    return `${url.protocol}//${url.host}/`;
  } catch {
    return "";
  }
}

function resolveHTTPSURL(value: string, baseURL: string): string {
  if (!value) return "";
  try {
    return normalizePublicHTTPSURL(new URL(value, baseURL).toString());
  } catch {
    return "";
  }
}

function privateHostname(hostname: string): boolean {
  const h = hostname.toLowerCase();
  if (h === "localhost" || h.endsWith(".localhost") || h.endsWith(".local")) return true;
  if (/^\d+\.\d+\.\d+\.\d+$/.test(h)) {
    const [a, b] = h.split(".").map((part) => Number(part));
    return a === 10 || a === 127 || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168) || a === 0 || a >= 224;
  }
  if (h === "::1" || h.startsWith("[::1]")) return true;
  return false;
}

function hostFromProviderBaseURL(baseURL: string): string {
  try {
    return new URL(baseURL).host.toLowerCase();
  } catch {
    return "";
  }
}

function officialProviderBaseURL(baseURL: string): boolean {
  return baseURL === "https://api.openai.com" || baseURL === "https://api.openai.com/v1";
}

function slugify(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_.-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
}

function decodeHTMLEntities(value: string): string {
  return value
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&#(\d+);/g, (_, n) => {
      const code = Number(n);
      return Number.isFinite(code) ? String.fromCodePoint(code) : "";
    });
}

function requiredString(v: unknown): boolean {
  return typeof v === "string" && v.trim() !== "" && v.length <= MAX_STRING_LENGTH;
}

function validInt(v: unknown, min: number, max: number): boolean {
  return typeof v === "number" && Number.isInteger(v) && v >= min && v <= max;
}

function validNumber(v: unknown, min: number, max: number): boolean {
  return typeof v === "number" && Number.isFinite(v) && v >= min && v <= max;
}

function stringArray(v: unknown, maxItems: number, maxItemLength: number): boolean {
  return Array.isArray(v) && v.length <= maxItems && v.every((item) => typeof item === "string" && item.length <= maxItemLength);
}

function clampInt(v: string | null, min: number, max: number, fallback: number): number {
  const n = Number(v);
  if (!Number.isInteger(n)) return fallback;
  return Math.min(max, Math.max(min, n));
}

function str(v: unknown, maxLength = MAX_STRING_LENGTH): string {
  const s = typeof v === "string" ? v : v == null ? "" : String(v);
  return s.slice(0, maxLength);
}

function jsonArrayString(v: unknown, maxLength: number): string {
  if (!Array.isArray(v)) return "[]";
  const items = v.map((item) => str(item, 128));
  while (items.length > 0) {
    const encoded = JSON.stringify(items);
    if (encoded.length <= maxLength) return encoded;
    items.pop();
  }
  return "[]";
}

function int(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? Math.trunc(n) : 0;
}

function num(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}
