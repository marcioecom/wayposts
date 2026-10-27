import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { ReactNode } from "react";

const en = {
  "brand.tagline": "A social feed under load",
  "brand.lab": "LAB",

  "nav.home": "Home",
  "nav.profile": "Profile",
  "nav.live": "Live",
  "nav.you": "You",
  "nav.newPost": "New post",
  "nav.primary": "Primary",
  "nav.language": "Language",
  "nav.version": "v1 · dev auth",
  "nav.openLive": "Open live metrics",
  "nav.openYou": "Open your identity",
  "nav.close": "Close",

  "status.connecting": "connecting",
  "status.open": "live",
  "status.closed": "offline",

  "tabs.label": "Timeline",
  "tabs.home": "Home",
  "tabs.profile": "Profile",
  "tabs.homeSub": "Posts from accounts you follow",
  "tabs.profileSub": "Posts from one account",

  "feed.refresh": "Refresh timeline",
  "feed.live": "Live",
  "feed.liveRate": "{rate} req/s",

  "composer.placeholder": "What is happening in the lab?",
  "composer.publish": "Publish",
  "composer.publishing": "Publishing",
  "composer.as": "Publishing as",
  "composer.hint": "Read-your-writes is on. Followers see it after fanout.",
  "composer.shortcut": "⌘↵ to publish",
  "composer.limit": "{count} / {max}",
  "composer.over": "Over the limit",

  "post.like": "Like",
  "post.unlike": "Unlike",
  "post.profile": "Profile",
  "post.delete": "Delete",
  "post.deleting": "Deleting",
  "post.deleteConfirm": "Delete post {id}?",
  "post.attachment": "Post attachment",
  "post.id": "Post ID {id}",
  "post.openProfile": "Open profile for user {id}",

  "time.justNow": "just now",
  "time.unknown": "unknown time",

  "empty.homeTitle": "Your home timeline is empty",
  "empty.homeCopy":
    "Follow an account, then publish from it. Fanout is asynchronous, so new posts arrive shortly after.",
  "empty.homeCta": "Follow by ID",
  "empty.profileTitle": "No posts from this account",
  "empty.profileCopy": "Publish from this identity or inspect another profile.",
  "empty.profileCta": "Switch identity",
  "empty.end": (count: number) =>
    `End of page · ${count} post${count === 1 ? "" : "s"}`,
  "empty.loading": "Loading timeline",
  "empty.loadMore": "Load more",
  "empty.loadingMore": "Loading",

  "error.timelineTitle": "Could not load the timeline",
  "error.network": "The API did not answer. Check the tunnel and CORS.",
  "error.user": "Could not load user {id}",
  "error.retry": "Try again",
  "error.positiveId": "Enter a positive user ID.",
  "error.followInvalid": "Enter a positive user ID to follow.",

  "nudge.new": (count: number) =>
    `${count} new post${count === 1 ? "" : "s"}`,
  "nudge.refresh": "Refresh",
  "nudge.dismiss": "Dismiss",
  "nudge.eyebrow": "Nudge",
  "nudge.listening": "Listening for new posts",
  "nudge.connecting": "Connecting to the gateway",
  "nudge.closed": "Disconnected, retrying",
  "nudge.hint":
    "The gateway sweeps materialized feeds every 30s. Celebrity posts merge on read and never nudge.",
  "nudge.refreshNow": "Refresh now",
  "nudge.head": "head {id}",

  "identity.eyebrow": "You",
  "identity.switch": "Switch identity",
  "identity.followers": (count: number) =>
    `${count} follower${count === 1 ? "" : "s"}`,
  "identity.openProfile": "Open profile",
  "identity.dialogTitle": "Switch identity",
  "identity.dialogCopy":
    "Dev auth uses the bearer token as the user ID. Seeds run from 1 to 2,000. Users 1 to 3 are celebrities.",
  "identity.label": "User ID",
  "identity.use": "Use identity",
  "identity.cancel": "Cancel",
  "identity.quick": "Quick identities",
  "identity.current": "Current",
  "identity.loading": "Loading user",
  "identity.notFound": "User not found",
  "identity.hint":
    "Switching identity resets the feed and reconnects the nudge socket.",

  "follow.eyebrow": "Graph",
  "follow.title": "Follow by ID",
  "follow.label": "User ID",
  "follow.follow": "Follow",
  "follow.unfollow": "Unfollow",
  "follow.open": "Open user {id}",
  "follow.hint":
    "Following affects future fanout only. Existing posts are not backfilled.",
  "follow.acting": "acting as @{handle}",

  "traffic.eyebrow": "Gateway load",
  "traffic.view": "metrics",
  "traffic.req": "req/s",
  "traffic.p95": "p95 · 5s",
  "traffic.errors": "5xx",
  "traffic.inflight": "in flight",
  "traffic.timeline": "timeline reads/s",
  "traffic.writes": "post writes/s",
  "traffic.likes": "likes/s",
  "traffic.window": "last 5 min",
  "traffic.updated": "updated {age}",
  "traffic.live": "live",
  "traffic.idle": "idle",
  "traffic.stale": "stale",
  "traffic.loading": "Reading gateway metrics…",
  "traffic.error": "Metrics unreachable",
  "traffic.errorHint": "The gateway did not answer {path}.",
  "traffic.retry": "Retry",
  "traffic.idleHint":
    "No load right now. Start the generator to populate this panel.",
  "traffic.command": "./scripts/load.sh steady",
  "traffic.copy": "Copy command",
  "traffic.copied": "Copied",
  "traffic.note":
    "Gateway view. /metrics and /healthz are excluded. Fanout lag is measured on the fanout service.",
  "traffic.justNow": "just now",
  "traffic.ago": "{seconds}s ago",

  "system.details": "System details",
  "system.api": "API",
  "system.posts": "Posts loaded",
  "system.cursor": "Next cursor",
  "system.more": "More pages",
  "system.health": "API health",
  "system.metrics": "Gateway metrics",
  "system.yes": "yes",
  "system.no": "no",
  "system.none": "none",
  "system.ws": "Nudge socket",

  "notes.summary": "Architecture notes",
  "notes.one": "Publishing returns before fanout reaches followers.",
  "notes.two": "Celebrity posts are merged on read, not materialized.",
  "notes.three": "Follow has no backfill. Only future posts reach the feed.",
  "notes.four": "Delete is lazy. Timelines drop deleted posts on read.",

  "profile.notFound": "User {id} was not found",
  "profile.notFoundCopy": "The social service has no user with this ID.",
  "profile.errorTitle": "Could not load this profile",
  "profile.follow": "Follow",
  "profile.unfollow": "Unfollow",
  "profile.current": "This is you.",
  "profile.currentHint":
    "Switch identity to act as someone else.",
  "profile.hint":
    "The API does not expose whether this follow already exists, so follow and unfollow are explicit actions.",
  "profile.viewing": "viewing as @{handle}",
  "profile.refresh": "Refresh profile",
  "profile.eyebrow": "Profile",
  "profile.posts": "{count} on page",

  "sheet.liveTitle": "Live traffic",
  "sheet.youTitle": "You",
  "sheet.close": "Close",

  "toast.dismiss": "Dismiss message",
  "toast.postPublished": "Post published. Fanout is running.",
  "toast.postDeleted": "Post deleted.",
  "toast.following": "Now following user {id}.",
  "toast.unfollowed": "Unfollowed user {id}.",

  "aria.timeline": "Timeline",
  "aria.language": "Language",
  "aria.quickIdentities": "Quick identities",
  "aria.nudgeStatus": "Nudge socket status",
  "aria.traffic": "Live gateway traffic",
  "aria.identity": "Current identity",
  "aria.sparkline": "Requests per second over the last five minutes",
} as const;

export type TranslationKey = keyof typeof en;

type Dictionary = {
  [K in TranslationKey]: string | ((count: number) => string);
};

const pt: Dictionary = {
  "brand.tagline": "Um feed social sob carga",
  "brand.lab": "LAB",

  "nav.home": "Home",
  "nav.profile": "Perfil",
  "nav.live": "Ao vivo",
  "nav.you": "Você",
  "nav.newPost": "Novo post",
  "nav.primary": "Principal",
  "nav.language": "Idioma",
  "nav.version": "v1 · auth de dev",
  "nav.openLive": "Abrir métricas ao vivo",
  "nav.openYou": "Abrir sua identidade",
  "nav.close": "Fechar",

  "status.connecting": "conectando",
  "status.open": "ao vivo",
  "status.closed": "offline",

  "tabs.label": "Timeline",
  "tabs.home": "Home",
  "tabs.profile": "Perfil",
  "tabs.homeSub": "Posts das contas que você segue",
  "tabs.profileSub": "Posts de uma conta",

  "feed.refresh": "Atualizar timeline",
  "feed.live": "Ao vivo",
  "feed.liveRate": "{rate} req/s",

  "composer.placeholder": "O que está acontecendo no laboratório?",
  "composer.publish": "Publicar",
  "composer.publishing": "Publicando",
  "composer.as": "Publicando como",
  "composer.hint":
    "Leitura das suas escritas está ligada. Seguidores veem depois do fanout.",
  "composer.shortcut": "⌘↵ para publicar",
  "composer.limit": "{count} / {max}",
  "composer.over": "Acima do limite",

  "post.like": "Curtir",
  "post.unlike": "Descurtir",
  "post.profile": "Perfil",
  "post.delete": "Deletar",
  "post.deleting": "Deletando",
  "post.deleteConfirm": "Deletar o post {id}?",
  "post.attachment": "Anexo do post",
  "post.id": "ID do post {id}",
  "post.openProfile": "Abrir perfil do usuário {id}",

  "time.justNow": "agora",
  "time.unknown": "horário desconhecido",

  "empty.homeTitle": "Sua home timeline está vazia",
  "empty.homeCopy":
    "Siga uma conta e publique por ela. O fanout é assíncrono, então os posts chegam um instante depois.",
  "empty.homeCta": "Seguir por ID",
  "empty.profileTitle": "Nenhum post desta conta",
  "empty.profileCopy": "Publique com esta identidade ou inspecione outro perfil.",
  "empty.profileCta": "Trocar identidade",
  "empty.end": (count: number) =>
    `Fim da página · ${count} post${count === 1 ? "" : "s"}`,
  "empty.loading": "Carregando timeline",
  "empty.loadMore": "Carregar mais",
  "empty.loadingMore": "Carregando",

  "error.timelineTitle": "Não foi possível carregar a timeline",
  "error.network": "A API não respondeu. Confira o túnel e o CORS.",
  "error.user": "Não foi possível carregar o usuário {id}",
  "error.retry": "Tentar de novo",
  "error.positiveId": "Informe um ID de usuário positivo.",
  "error.followInvalid": "Informe um ID de usuário positivo para seguir.",

  "nudge.new": (count: number) =>
    `${count} post${count === 1 ? "" : "s"} novo${count === 1 ? "" : "s"}`,
  "nudge.refresh": "Atualizar",
  "nudge.dismiss": "Dispensar",
  "nudge.eyebrow": "Nudge",
  "nudge.listening": "Ouvindo novos posts",
  "nudge.connecting": "Conectando ao gateway",
  "nudge.closed": "Desconectado, tentando de novo",
  "nudge.hint":
    "O gateway varre os feeds materializados a cada 30s. Posts de celebridade são lidos no merge e não geram nudge.",
  "nudge.refreshNow": "Atualizar agora",
  "nudge.head": "head {id}",

  "identity.eyebrow": "Você",
  "identity.switch": "Trocar identidade",
  "identity.followers": (count: number) =>
    `${count} seguidor${count === 1 ? "" : "es"}`,
  "identity.openProfile": "Abrir perfil",
  "identity.dialogTitle": "Trocar identidade",
  "identity.dialogCopy":
    "Auth de dev usa o bearer token como ID do usuário. O seed vai de 1 a 2.000. Usuários 1 a 3 são celebridades.",
  "identity.label": "ID do usuário",
  "identity.use": "Usar identidade",
  "identity.cancel": "Cancelar",
  "identity.quick": "Identidades rápidas",
  "identity.current": "Atual",
  "identity.loading": "Carregando usuário",
  "identity.notFound": "Usuário não encontrado",
  "identity.hint":
    "Trocar de identidade reinicia o feed e reconecta o socket de nudge.",

  "follow.eyebrow": "Grafo",
  "follow.title": "Seguir por ID",
  "follow.label": "ID do usuário",
  "follow.follow": "Seguir",
  "follow.unfollow": "Deixar de seguir",
  "follow.open": "Abrir usuário {id}",
  "follow.hint":
    "Seguir afeta só o fanout futuro. Posts existentes não são backfilled.",
  "follow.acting": "agindo como @{handle}",

  "traffic.eyebrow": "Carga do gateway",
  "traffic.view": "métricas",
  "traffic.req": "req/s",
  "traffic.p95": "p95 · 5s",
  "traffic.errors": "5xx",
  "traffic.inflight": "em andamento",
  "traffic.timeline": "leituras de timeline/s",
  "traffic.writes": "posts escritos/s",
  "traffic.likes": "likes/s",
  "traffic.window": "últimos 5 min",
  "traffic.updated": "atualizado {age}",
  "traffic.live": "ao vivo",
  "traffic.idle": "ocioso",
  "traffic.stale": "desatualizado",
  "traffic.loading": "Lendo métricas do gateway…",
  "traffic.error": "Métricas indisponíveis",
  "traffic.errorHint": "O gateway não respondeu em {path}.",
  "traffic.retry": "Tentar de novo",
  "traffic.idleHint":
    "Sem carga agora. Inicie o gerador para preencher este painel.",
  "traffic.command": "./scripts/load.sh steady",
  "traffic.copy": "Copiar comando",
  "traffic.copied": "Copiado",
  "traffic.note":
    "Visão do gateway. /metrics e /healthz ficam fora. O fanout lag é medido no serviço de fanout.",
  "traffic.justNow": "agora",
  "traffic.ago": "{seconds}s atrás",

  "system.details": "Detalhes do sistema",
  "system.api": "API",
  "system.posts": "Posts carregados",
  "system.cursor": "Próximo cursor",
  "system.more": "Mais páginas",
  "system.health": "Saúde da API",
  "system.metrics": "Métricas do gateway",
  "system.yes": "sim",
  "system.no": "não",
  "system.none": "nenhum",
  "system.ws": "Socket de nudge",

  "notes.summary": "Notas de arquitetura",
  "notes.one": "Publicar retorna antes do fanout alcançar os seguidores.",
  "notes.two": "Posts de celebridade são lidos no merge, não materializados.",
  "notes.three": "Follow não faz backfill. Só posts futuros chegam ao feed.",
  "notes.four": "Delete é preguiçoso. Timelines descartam posts deletados na leitura.",

  "profile.notFound": "Usuário {id} não encontrado",
  "profile.notFoundCopy": "O serviço social não tem usuário com este ID.",
  "profile.errorTitle": "Não foi possível carregar este perfil",
  "profile.follow": "Seguir",
  "profile.unfollow": "Deixar de seguir",
  "profile.current": "Este é você.",
  "profile.currentHint":
    "Troque de identidade para agir como outra pessoa.",
  "profile.hint":
    "A API não expõe se este follow já existe, então seguir e deixar de seguir são ações explícitas.",
  "profile.viewing": "vendo como @{handle}",
  "profile.refresh": "Atualizar perfil",
  "profile.eyebrow": "Perfil",
  "profile.posts": "{count} na página",

  "sheet.liveTitle": "Tráfego ao vivo",
  "sheet.youTitle": "Você",
  "sheet.close": "Fechar",

  "toast.dismiss": "Dispensar mensagem",
  "toast.postPublished": "Post publicado. O fanout está rodando.",
  "toast.postDeleted": "Post deletado.",
  "toast.following": "Agora seguindo o usuário {id}.",
  "toast.unfollowed": "Deixou de seguir o usuário {id}.",

  "aria.timeline": "Timeline",
  "aria.language": "Idioma",
  "aria.quickIdentities": "Identidades rápidas",
  "aria.nudgeStatus": "Status do socket de nudge",
  "aria.traffic": "Tráfego ao vivo do gateway",
  "aria.identity": "Identidade atual",
  "aria.sparkline": "Requisições por segundo nos últimos cinco minutos",
};

export type Lang = "en" | "pt";

const dictionaries: Record<Lang, Dictionary> = { en, pt };

const LANGUAGE_KEY = "wayposts.web.lang";

export function localeFor(lang: Lang): string {
  return lang === "pt" ? "pt-BR" : "en-US";
}

function readLang(): Lang {
  try {
    const saved = localStorage.getItem(LANGUAGE_KEY);
    if (saved === "en" || saved === "pt") return saved;
  } catch {
    // Storage can be unavailable in private mode.
  }
  if (typeof navigator !== "undefined") {
    return navigator.language.toLowerCase().startsWith("pt") ? "pt" : "en";
  }
  return "en";
}

interface I18nValue {
  lang: Lang;
  locale: string;
  setLang: (lang: Lang) => void;
  t: (
    key: TranslationKey,
    vars?: Record<string, string | number>,
  ) => string;
}

const I18nContext = createContext<I18nValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(readLang);

  useEffect(() => {
    document.documentElement.lang = lang === "pt" ? "pt-BR" : "en";
    document.title =
      lang === "pt"
        ? "Wayposts Lab. Um feed social sob carga"
        : "Wayposts Lab. A social feed under load";
  }, [lang]);

  const setLang = useCallback((next: Lang) => {
    setLangState(next);
    try {
      localStorage.setItem(LANGUAGE_KEY, next);
    } catch {
      // The in-memory language still works.
    }
  }, []);

  const t = useCallback<I18nValue["t"]>(
    (key, vars) => {
      const value = dictionaries[lang][key] ?? en[key];
      const text =
        typeof value === "function"
          ? value(Number(vars?.count ?? 0))
          : value;
      if (!vars) return text;
      return text.replace(/\{(\w+)\}/g, (match, name: string) =>
        name in vars ? String(vars[name]) : match,
      );
    },
    [lang],
  );

  const value = useMemo(
    () => ({ lang, locale: localeFor(lang), setLang, t }),
    [lang, setLang, t],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const value = useContext(I18nContext);
  if (!value) throw new Error("useI18n must be used inside I18nProvider");
  return value;
}
