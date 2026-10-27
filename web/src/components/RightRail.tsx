import type { WsStatus, User } from "../types";
import type { TrafficState } from "../lib/metrics";
import { FollowCard } from "./FollowCard";
import { NudgeCard } from "./NudgeCard";
import { ArchitectureNotes, SystemDetails } from "./SystemDetails";
import { TrafficPanel } from "./TrafficPanel";

export interface LivePanelsProps {
  traffic: TrafficState & { retry: () => void };
  apiBase: string;
  wsStatus: WsStatus;
  nudgeCount: number | null;
  nudgeHead: number | null;
  onRefresh: () => void;
  currentUserId: number;
  currentUser: User | null;
  onFollow: (followeeId: number, action: "follow" | "unfollow") => void;
  onOpenProfile: (id: number) => void;
  postCount: number;
  nextCursor: number;
  hasMore: boolean;
}

export function LivePanels({
  traffic,
  apiBase,
  wsStatus,
  nudgeCount,
  nudgeHead,
  onRefresh,
  currentUserId,
  currentUser,
  onFollow,
  onOpenProfile,
  postCount,
  nextCursor,
  hasMore,
}: LivePanelsProps) {
  return (
    <div className="live-panels">
      <TrafficPanel traffic={traffic} apiBase={apiBase} />
      <NudgeCard
        status={wsStatus}
        count={nudgeCount}
        head={nudgeHead}
        onRefresh={onRefresh}
      />
      <FollowCard
        currentUserId={currentUserId}
        currentHandle={currentUser?.handle ?? String(currentUserId)}
        onFollow={onFollow}
        onOpenProfile={onOpenProfile}
      />
      <SystemDetails
        apiBase={apiBase}
        wsStatus={wsStatus}
        postCount={postCount}
        nextCursor={nextCursor}
        hasMore={hasMore}
      />
      <ArchitectureNotes />
    </div>
  );
}

export function RightRail(props: LivePanelsProps) {
  return (
    <aside className="rail-right">
      <div className="rail-right-inner">
        <LivePanels {...props} />
      </div>
    </aside>
  );
}
