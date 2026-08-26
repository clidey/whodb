package com.clidey.whodb;

import com.clidey.whodb.gen.Manifest;
import com.clidey.whodb.gen.Operations;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;

/**
 * The WhoDB platform client. Credential precedence: explicit {@link Config} →
 * {@code WHODB_API_KEY} env → the whodb CLI's stored login; inside the
 * Functions runtime the IPC transport is auto-detected via
 * {@code WHODB_IPC_TOKEN}.
 */
public final class WhoDB {

    /** SDK release version, stamped by sync-versions.mjs; feeds User-Agent. */
    public static final String SDK_VERSION = "0.0.0";

    /** The hosted WhoDB platform endpoint. */
    public static final String DEFAULT_HOST = "https://app.whodb.com";

    private static final Pattern UUID_PATTERN =
        Pattern.compile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$");

    /**
     * Client configuration. All fields are optional; unset fields fall back
     * to the standard precedence.
     *
     * @param apiKey platform API key ({@code whodb_sk_...}); highest-precedence credential
     * @param token raw OIDC access token
     * @param credentials fully custom provider (overrides apiKey/token)
     * @param org organization slug or ID; optional with an API key
     * @param project project slug or ID; optional with a single-grant API key
     * @param host platform endpoint override
     * @param transport wire transport override (IPC, tests); org/project are taken as IDs verbatim
     */
    public record Config(String apiKey, String token, CredentialProvider credentials,
                         String org, String project, String host, Transport transport) {
        /** An all-defaults configuration. */
        public static Config defaults() {
            return new Config(null, null, null, null, null, null, null);
        }
    }

    private final Transport transport;
    private final HttpTransport http; // null for custom transports
    private final CliCredentials cli;
    private final boolean usingApiKey;
    private final boolean skipResolution;
    private final String orgInput;
    private final String projectInput;

    private boolean workspaceResolved;
    private String resolvedProject;

    /** Creates a client with all-default configuration. */
    public WhoDB() {
        this(Config.defaults());
    }

    /** Creates a client, applying the credential precedence. */
    public WhoDB(Config config) {
        this.orgInput = firstNonEmpty(config.org(), System.getenv("WHODB_ORG"));
        this.projectInput = firstNonEmpty(config.project(), System.getenv("WHODB_PROJECT"));
        String envApiKey = System.getenv("WHODB_API_KEY");
        Transport chosen = config.transport();
        if (chosen == null && config.credentials() == null && isEmpty(config.apiKey()) && isEmpty(config.token())
            && isEmpty(envApiKey) && !isEmpty(System.getenv("WHODB_IPC_TOKEN"))) {
            chosen = new IpcTransport(IpcTransport.Config.fromEnv());
        }
        if (chosen != null) {
            this.transport = chosen;
            this.http = null;
            this.cli = null;
            this.usingApiKey = false;
            this.skipResolution = true;
            return;
        }
        CredentialProvider credentials;
        boolean apiKey = false;
        CliCredentials cliProvider = null;
        if (config.credentials() != null) {
            credentials = config.credentials();
        } else if (!isEmpty(config.apiKey())) {
            String value = config.apiKey();
            credentials = () -> value;
            apiKey = true;
        } else if (!isEmpty(config.token())) {
            String value = config.token();
            credentials = () -> value;
        } else if (!isEmpty(envApiKey)) {
            credentials = () -> envApiKey;
            apiKey = true;
        } else {
            cliProvider = new CliCredentials();
            credentials = cliProvider;
        }
        this.cli = cliProvider;
        this.usingApiKey = apiKey;
        this.skipResolution = false;
        String host = firstNonEmpty(config.host(), System.getenv("WHODB_HOST"), DEFAULT_HOST);
        this.http = new HttpTransport(host, credentials);
        this.transport = this.http;
    }

    private static boolean isEmpty(String value) {
        return value == null || value.isEmpty();
    }

    private static String firstNonEmpty(String... values) {
        for (String value : values) {
            if (!isEmpty(value)) {
                return value;
            }
        }
        return "";
    }

    /**
     * The SHA-256 of the platform manifest this SDK release was generated
     * from.
     */
    public static String manifestHash() {
        return Manifest.MANIFEST_HASH;
    }

    /**
     * Runs one prepared operation and unwraps its payload, converting
     * unknown-operation rejections into the version-upgrade error.
     */
    Object execute(Operations.Request request) {
        Map<String, Object> data;
        try {
            data = transport.execute(request.operation(), request.document(), request.variables());
        } catch (WhoDBException error) {
            throw WhoDBException.interpretServerError(error);
        }
        return data.get(request.operation());
    }

    /**
     * Resolves org/project (slug or ID) to IDs once, stamping the workspace
     * headers onto the HTTP transport. API keys may omit both — the platform
     * discovers them via MyWorkspace.
     */
    @SuppressWarnings("unchecked")
    private synchronized void resolveWorkspace() {
        if (workspaceResolved) {
            return;
        }
        String org = orgInput;
        String project = projectInput;
        if ((org.isEmpty() || project.isEmpty()) && cli != null) {
            try {
                CliCredentials.TokenEntry defaults = cli.defaults();
                org = firstNonEmpty(org, defaults.orgId());
                project = firstNonEmpty(project, defaults.projectId());
            } catch (WhoDBException ignored) {
                // fall through to the explicit-config error below
            }
        }
        if ((org.isEmpty() || project.isEmpty()) && usingApiKey) {
            ManifestCheck.warnIfFlagged("MyWorkspace");
            Object result = execute(Operations.myWorkspaceRequest(Map.of()));
            if (result instanceof Map<?, ?> workspace) {
                org = firstNonEmpty(org, stringOf(workspace.get("orgId")));
                project = firstNonEmpty(project, stringOf(workspace.get("projectId")));
            }
            if (project.isEmpty()) {
                throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                    "this API key has access to multiple (or zero) projects — set project in Config or WHODB_PROJECT");
            }
        }
        if (skipResolution) {
            resolvedProject = project;
            workspaceResolved = true;
            return;
        }
        if (org.isEmpty() || project.isEmpty()) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                "org and project are required — set them in Config, WHODB_ORG/WHODB_PROJECT, or run: whodb use");
        }
        String orgId = org;
        if (!UUID_PATTERN.matcher(org).matches()) {
            Object result = execute(Operations.myOrganizationsRequest(Map.of()));
            orgId = matchWorkspaceEntry(result, org);
            if (orgId.isEmpty()) {
                throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                    "organization \"" + org + "\" not found for this account");
            }
        }
        http.setWorkspace(orgId, "");
        String projectId = project;
        if (!UUID_PATTERN.matcher(project).matches()) {
            Object result = execute(Operations.projectsRequest(Map.of("orgId", orgId)));
            projectId = matchWorkspaceEntry(result, project);
            if (projectId.isEmpty()) {
                throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                    "project \"" + project + "\" not found in this organization");
            }
        }
        http.setWorkspace(orgId, projectId);
        resolvedProject = projectId;
        workspaceResolved = true;
    }

    private static String stringOf(Object value) {
        return value instanceof String text ? text : "";
    }

    /** Finds a workspace entry by slug or name and returns its id. */
    private static String matchWorkspaceEntry(Object result, String input) {
        if (result instanceof List<?> entries) {
            for (Object entry : entries) {
                if (entry instanceof Map<?, ?> object
                    && (input.equals(object.get("slug")) || input.equals(object.get("name")))) {
                    Object id = object.get("id");
                    return id instanceof String text ? text : "";
                }
            }
        }
        return "";
    }

    synchronized String projectId() {
        resolveWorkspace();
        return resolvedProject;
    }

    /** Returns a handle for one ontology entity, addressed by apiName. */
    public OntologyHandle ontology(String apiName) {
        return new OntologyHandle(this, apiName);
    }

    /** Lists all ontology entities in the project. */
    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> ontologyEntities() {
        String projectId = projectId();
        ManifestCheck.warnIfFlagged("OntologyEntities");
        Object result = execute(Operations.ontologyEntitiesRequest(Map.of("projectId", projectId)));
        List<Map<String, Object>> entities = new ArrayList<>();
        if (result instanceof List<?> entries) {
            for (Object entry : entries) {
                if (entry instanceof Map<?, ?> entity) {
                    entities.add((Map<String, Object>) entity);
                }
            }
        }
        return entities;
    }
}
