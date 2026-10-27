import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError, apiBase, errorMessage, nudgeUrl } from "./api";
import type { Post, ToastMessage, User, WsStatus } from "./types";
import { useI18n } from "./lib/i18n";
import { useTraffic } from "./lib/metrics";
import { Composer } from "./components/Composer";
import { FeedHeader } from "./components/FeedHeader";
import { LeftRail } from "./components/LeftRail";
import { BottomNav, Topbar } from "./components/MobileChrome";
import { ProfileHeader } from "./components/ProfileHeader";
import { RightRail } from "./components/RightRail";
import type { LivePanelsProps } from "./components/RightRail";
import { IdentityDialog, LiveSheet, YouSheet } from "./components/Sheets";
import { Timeline } from "./components/Timeline";
import { Toast } from "./components/Toast";

const PAGE_SIZE = 20;
const USER_KEY = "wayposts.web.userId";

interface TimelineState {
  items: Post[];
  nextCursor: number;
  hasMore: boolean;
}

function emptyTimeline(): TimelineState {
  return { items: [], nextCursor: 0, hasMore: false };
}

function readUserId(): number {
  try {
    const raw = localStorage.getItem(USER_KEY);
    const parsed = raw ? Number(raw) : 10;
    return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 10;
  } catch {
    return 10;
  }
}

export default function App() {
  const { t } = useI18n();
  const traffic = useTraffic(apiBase);

  const [currentUserId, setCurrentUserId] = useState<number>(readUserId);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [currentUserLoading, setCurrentUserLoading] = useState(true);
  const [userIdInput, setUserIdInput] = useState(String(readUserId()));

  const [tab, setTab] = useState<"home" | "profile">("home");
  const [profileUserId, setProfileUserId] = useState(currentUserId);
  const [profileUser, setProfileUser] = useState<User | null>(null);
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileError, setProfileError] = useState<{
    message: string;
    notFound: boolean;
  } | null>(null);

  const [timeline, setTimeline] = useState<TimelineState>(emptyTimeline);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshNonce, setRefreshNonce] = useState(0);

  const [users, setUsers] = useState<Record<number, User>>({});
  const requestedUsers = useRef(new Set<number>());
  const lastSeenRef = useRef(0);

  const [nudge, setNudge] = useState<{ count: number; head: number } | null>(null);
  const [wsStatus, setWsStatus] = useState<WsStatus>("closed");
  const [toast, setToast] = useState<ToastMessage | null>(null);

  const [identityOpen, setIdentityOpen] = useState(false);
  const [liveOpen, setLiveOpen] = useState(false);
  const [youOpen, setYouOpen] = useState(false);
  const [highlightedPostId, setHighlightedPostId] = useState<number | null>(null);

  const composerRef = useRef<HTMLTextAreaElement>(null);
  const highlightTimer = useRef<number | undefined>(undefined);

  const notify = useCallback((kind: "success" | "error", message: string) => {
    setToast({ id: Date.now(), kind, message });
  }, []);

  const markHighlight = useCallback((id: number) => {
    setHighlightedPostId(id);
    if (highlightTimer.current) window.clearTimeout(highlightTimer.current);
    highlightTimer.current = window.setTimeout(() => {
      setHighlightedPostId(null);
    }, 2600);
  }, []);

  useEffect(() => {
    return () => {
      if (highlightTimer.current) window.clearTimeout(highlightTimer.current);
    };
  }, []);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(null), 5000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    let cancelled = false;
    setCurrentUserLoading(true);
    setCurrentUser(null);
    api
      .getUser(currentUserId, currentUserId)
      .then((user) => {
        if (cancelled) return;
        setCurrentUser(user);
        setUsers((prev) => ({ ...prev, [user.id]: user }));
      })
      .catch((cause) => {
        if (!cancelled) {
          const raw = errorMessage(cause);
          const message =
            cause instanceof ApiError && cause.status === 404
              ? t("identity.notFound")
              : raw.startsWith("Cannot reach the API")
                ? t("error.network")
                : `${t("error.user", { id: currentUserId })}: ${raw}`;
          notify("error", message);
        }
      })
      .finally(() => {
        if (!cancelled) setCurrentUserLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [currentUserId, notify, t]);

  useEffect(() => {
    if (tab !== "profile") return;
    let cancelled = false;
    setProfileLoading(true);
    setProfileUser(null);
    setProfileError(null);
    api
      .getUser(currentUserId, profileUserId)
      .then((user) => {
        if (cancelled) return;
        setProfileUser(user);
        setUsers((prev) => ({ ...prev, [user.id]: user }));
      })
      .catch((cause) => {
        if (cancelled) return;
        const notFound = cause instanceof ApiError && cause.status === 404;
        setProfileError({ message: errorMessage(cause), notFound });
      })
      .finally(() => {
        if (!cancelled) setProfileLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tab, profileUserId, currentUserId, refreshNonce]);

  useEffect(() => {
    const ids = new Set<number>([currentUserId, profileUserId]);
    for (const item of timeline.items) ids.add(item.author_id);

    for (const id of ids) {
      if (users[id] || requestedUsers.current.has(id)) continue;
      requestedUsers.current.add(id);
      api
        .getUser(currentUserId, id)
        .then((user) => setUsers((prev) => ({ ...prev, [user.id]: user })))
        .catch(() => {
          // Author details are decorative. The post still renders with the ID.
        })
        .finally(() => requestedUsers.current.delete(id));
    }
  }, [currentUserId, profileUserId, timeline.items, users]);

  const fetchPage = useCallback(
    (before: number) => {
      return tab === "home"
        ? api.home(currentUserId, before, PAGE_SIZE)
        : api.profile(currentUserId, profileUserId, before, PAGE_SIZE);
    },
    [tab, currentUserId, profileUserId],
  );

  const loadReset = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const page = await fetchPage(0);
      setTimeline({
        items: page.items,
        nextCursor: page.next_cursor,
        hasMore: page.items.length >= PAGE_SIZE,
      });
      setNudge(null);
      if (tab === "home" && page.items[0]) {
        lastSeenRef.current = page.items[0].id;
      }
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setLoading(false);
    }
  }, [fetchPage, tab]);

  useEffect(() => {
    void loadReset();
  }, [loadReset, refreshNonce]);

  const loadMore = useCallback(async () => {
    if (loadingMore || !timeline.hasMore) return;
    setLoadingMore(true);
    setError(null);
    try {
      const page = await fetchPage(timeline.nextCursor);
      setTimeline((prev) => {
        const known = new Set(prev.items.map((item) => item.id));
        return {
          items: [...prev.items, ...page.items.filter((item) => !known.has(item.id))],
          nextCursor: page.next_cursor,
          hasMore: page.items.length >= PAGE_SIZE,
        };
      });
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setLoadingMore(false);
    }
  }, [fetchPage, loadingMore, timeline.hasMore, timeline.nextCursor]);

  useEffect(() => {
    if (tab !== "home") return;
    const head = timeline.items[0]?.id ?? 0;
    if (head > lastSeenRef.current) lastSeenRef.current = head;
  }, [tab, timeline.items]);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let disposed = false;
    let retryTimer: number | undefined;
    let attempt = 0;

    const connect = () => {
      if (disposed) return;
      setWsStatus("connecting");
      try {
        socket = new WebSocket(nudgeUrl(currentUserId, lastSeenRef.current));
      } catch {
        setWsStatus("closed");
        return;
      }

      socket.onopen = () => {
        attempt = 0;
        setWsStatus("open");
      };

      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as {
            type?: string;
            count?: number;
            head?: number;
          };
          if (payload.type === "new_posts" && typeof payload.count === "number") {
            setNudge({ count: payload.count, head: payload.head ?? 0 });
          }
        } catch {
          // Ignore malformed frames.
        }
      };

      socket.onerror = () => {
        socket?.close();
      };

      socket.onclose = () => {
        setWsStatus("closed");
        if (disposed) return;
        attempt += 1;
        const delay = Math.min(15_000, 1_000 * 2 ** Math.min(attempt, 4));
        retryTimer = window.setTimeout(connect, delay);
      };
    };

    connect();

    return () => {
      disposed = true;
      if (retryTimer) window.clearTimeout(retryTimer);
      socket?.close();
    };
  }, [currentUserId]);

  const closeSheets = useCallback(() => {
    setIdentityOpen(false);
    setLiveOpen(false);
    setYouOpen(false);
  }, []);

  const switchUser = useCallback(
    (id: number) => {
      const next = Math.floor(id);
      if (!Number.isFinite(next) || next <= 0) {
        notify("error", t("error.positiveId"));
        return;
      }
      try {
        localStorage.setItem(USER_KEY, String(next));
      } catch {
        // Storage can be unavailable in private mode. The in-memory user still works.
      }
      lastSeenRef.current = 0;
      setCurrentUserId(next);
      setUserIdInput(String(next));
      setProfileUserId(next);
      setTab("home");
      setTimeline(emptyTimeline());
      setProfileUser(null);
      setProfileError(null);
      setNudge(null);
      setError(null);
      setLoading(true);
      setRefreshNonce((value) => value + 1);
      closeSheets();
    },
    [closeSheets, notify, t],
  );

  const selectTab = useCallback((next: "home" | "profile") => {
    setTab(next);
    setNudge(null);
    setError(null);
    setTimeline(emptyTimeline());
    setLoading(true);
    setProfileError(null);
    if (next === "profile") {
      setProfileUser(null);
      setProfileLoading(true);
    }
  }, []);

  const openProfile = useCallback(
    (id: number) => {
      setProfileUserId(id);
      setTab("profile");
      setProfileUser(null);
      setProfileError(null);
      setProfileLoading(true);
      setNudge(null);
      setError(null);
      setTimeline(emptyTimeline());
      setLoading(true);
      setRefreshNonce((value) => value + 1);
      closeSheets();
      window.scrollTo({ top: 0 });
    },
    [closeSheets],
  );

  const openLive = useCallback(() => {
    setLiveOpen(true);
    setYouOpen(false);
    setIdentityOpen(false);
  }, []);

  const openYou = useCallback(() => {
    setYouOpen(true);
    setLiveOpen(false);
    setIdentityOpen(false);
  }, []);

  const openIdentity = useCallback(() => {
    setIdentityOpen(true);
    setLiveOpen(false);
    setYouOpen(false);
  }, []);

  const handleNewPost = useCallback(() => {
    if (tab !== "home") {
      selectTab("home");
    } else {
      setNudge(null);
    }
    closeSheets();
    window.scrollTo({ top: 0 });
    window.setTimeout(() => composerRef.current?.focus(), 80);
  }, [closeSheets, selectTab, tab]);

  const handleRefresh = useCallback(() => {
    setRefreshNonce((value) => value + 1);
  }, []);

  const handleCreate = useCallback(
    async (body: string): Promise<boolean> => {
      setCreating(true);
      try {
        const created = await api.createPost(currentUserId, body);
        setTimeline((prev) => ({
          ...prev,
          items: [created, ...prev.items.filter((item) => item.id !== created.id)],
        }));
        setTab("home");
        markHighlight(created.id);
        notify("success", t("toast.postPublished"));
        return true;
      } catch (cause) {
        notify("error", errorMessage(cause));
        return false;
      } finally {
        setCreating(false);
      }
    },
    [currentUserId, markHighlight, notify, t],
  );

  const handleLike = useCallback(
    async (post: Post) => {
      try {
        const result = post.viewer_liked
          ? await api.unlike(currentUserId, post.id)
          : await api.like(currentUserId, post.id);
        setTimeline((prev) => ({
          ...prev,
          items: prev.items.map((item) =>
            item.id === post.id
              ? { ...item, like_count: result.like_count, viewer_liked: result.liked }
              : item,
          ),
        }));
      } catch (cause) {
        notify("error", errorMessage(cause));
      }
    },
    [currentUserId, notify],
  );

  const handleDelete = useCallback(
    async (post: Post) => {
      if (!window.confirm(t("post.deleteConfirm", { id: post.id }))) return;
      try {
        await api.deletePost(currentUserId, post.id);
        setTimeline((prev) => ({
          ...prev,
          items: prev.items.filter((item) => item.id !== post.id),
        }));
        notify("success", t("toast.postDeleted"));
      } catch (cause) {
        notify("error", errorMessage(cause));
      }
    },
    [currentUserId, notify, t],
  );

  const handleFollow = useCallback(
    async (followeeId: number, action: "follow" | "unfollow") => {
      try {
        if (action === "follow") {
          await api.follow(currentUserId, followeeId);
          notify("success", t("toast.following", { id: followeeId }));
        } else {
          await api.unfollow(currentUserId, followeeId);
          notify("success", t("toast.unfollowed", { id: followeeId }));
        }
        setRefreshNonce((value) => value + 1);
      } catch (cause) {
        notify("error", errorMessage(cause));
      }
    },
    [currentUserId, notify, t],
  );

  const livePanels: LivePanelsProps = {
    traffic,
    apiBase,
    wsStatus,
    nudgeCount: nudge?.count ?? null,
    nudgeHead: nudge?.head ?? null,
    onRefresh: handleRefresh,
    currentUserId,
    currentUser,
    onFollow: handleFollow,
    onOpenProfile: openProfile,
    postCount: timeline.items.length,
    nextCursor: timeline.nextCursor,
    hasMore: timeline.hasMore,
  };

  const handleEmptyAction = useCallback(() => {
    if (tab === "home") {
      openLive();
    } else {
      openIdentity();
    }
  }, [openIdentity, openLive, tab]);

  const profileUnavailable =
    tab === "profile" && !profileLoading && !profileUser;
  const timelineUnavailable =
    profileUnavailable || (error !== null && timeline.items.length === 0);
  const timelineError =
    error && error.startsWith("Cannot reach the API")
      ? t("error.network")
      : error;

  const selectTabAndScroll = useCallback(
    (next: "home" | "profile") => {
      selectTab(next);
      window.scrollTo({ top: 0 });
    },
    [selectTab],
  );

  return (
    <div className="app-shell">
      <Topbar
        currentUserId={currentUserId}
        currentUser={currentUser}
        loading={currentUserLoading}
        traffic={traffic}
        onOpenLive={openLive}
        onOpenYou={openYou}
        onRefresh={handleRefresh}
      />

      <div className="app">
        <LeftRail
          tab={tab}
          currentUserId={currentUserId}
          currentUser={currentUser}
          loading={currentUserLoading}
          onSelectTab={selectTabAndScroll}
          onNewPost={handleNewPost}
          onOpenIdentity={openIdentity}
          onOpenLive={openLive}
        />

        <main className="feed">
          <FeedHeader
            tab={tab}
            profileUser={tab === "profile" ? profileUser : null}
            loading={loading}
            traffic={traffic}
            onSelectTab={selectTabAndScroll}
            onRefresh={handleRefresh}
            onOpenLive={openLive}
          />

          {tab === "home" ? (
            <Composer
              currentUserId={currentUserId}
              currentUser={currentUser}
              creating={creating}
              textareaRef={composerRef}
              onCreate={handleCreate}
            />
          ) : (
            <ProfileHeader
              userId={profileUserId}
              user={profileUser}
              loading={profileLoading}
              error={profileError}
              currentUserId={currentUserId}
              currentUser={currentUser}
              onFollow={handleFollow}
              onOpenIdentity={openIdentity}
              onRefresh={handleRefresh}
            />
          )}

          {nudge ? (
            <div className="nudge-banner" role="status">
              <div className="nudge-banner-copy">
                <span className="dot" aria-hidden="true" />
                <strong>{t("nudge.new", { count: nudge.count })}</strong>
                {nudge.head ? (
                  <span className="mono">{t("nudge.head", { id: nudge.head })}</span>
                ) : null}
              </div>
              <div className="nudge-banner-actions">
                <button
                  type="button"
                  className="btn btn-accent btn-sm"
                  onClick={() => void loadReset()}
                >
                  {t("nudge.refresh")}
                </button>
                <button
                  type="button"
                  className="btn btn-sm"
                  onClick={() => setNudge(null)}
                >
                  {t("nudge.dismiss")}
                </button>
              </div>
            </div>
          ) : null}

          {error && !profileUnavailable ? (
            <div className="error-block" role="alert">
              <div>
                <strong>{t("error.timelineTitle")}</strong>
                <p>{timelineError}</p>
              </div>
              <button
                type="button"
                className="btn btn-sm"
                onClick={() => void loadReset()}
              >
                {t("error.retry")}
              </button>
            </div>
          ) : null}

          {timelineUnavailable ? null : (
            <Timeline
              tab={tab}
              items={timeline.items}
              users={users}
              currentUserId={currentUserId}
              loading={loading}
              loadingMore={loadingMore}
              hasMore={timeline.hasMore}
              highlightedPostId={highlightedPostId}
              onLoadMore={() => void loadMore()}
              onLike={handleLike}
              onDelete={handleDelete}
              onOpenProfile={openProfile}
              onEmptyAction={handleEmptyAction}
            />
          )}
        </main>

        <RightRail {...livePanels} />
      </div>

      <BottomNav
        tab={tab}
        currentUserId={currentUserId}
        currentUser={currentUser}
        onSelectTab={selectTabAndScroll}
        onOpenLive={openLive}
        onOpenYou={openYou}
      />

      {toast ? <Toast toast={toast} onClose={() => setToast(null)} /> : null}

      <LiveSheet open={liveOpen} onClose={() => setLiveOpen(false)} {...livePanels} />

      <YouSheet
        open={youOpen}
        onClose={() => setYouOpen(false)}
        currentUserId={currentUserId}
        currentUser={currentUser}
        currentUserLoading={currentUserLoading}
        userIdInput={userIdInput}
        onInput={setUserIdInput}
        onSwitch={switchUser}
        onOpenProfile={openProfile}
      />

      <IdentityDialog
        open={identityOpen}
        onClose={() => setIdentityOpen(false)}
        currentUserId={currentUserId}
        currentUser={currentUser}
        currentUserLoading={currentUserLoading}
        userIdInput={userIdInput}
        onInput={setUserIdInput}
        onSwitch={switchUser}
        onOpenProfile={openProfile}
      />
    </div>
  );
}
