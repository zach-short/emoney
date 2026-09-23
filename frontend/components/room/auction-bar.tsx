import { Player, Property, Room } from "@/types/schema";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import { josephinBold, josephinNormal } from "../ui/fonts";
import AuctionPanel from "./auction-panel";

// The strip that says an auction is running, and the sheet it opens.
//
// It renders nothing at all when `room.auction` is absent, which is the normal
// state of every room -- Go tags the field `omitempty` on a pointer, so "no
// auction" arrives as a missing key rather than an empty object.
//
// The strip lives in the room header rather than floating over the cards: the
// header is already sticky, and a fixed bar at the bottom of the viewport would
// sit on top of each card's own "Pay or Request" button at phone width.
//
// It is deliberately not auto-opened when an auction starts. AUCTION_STARTED
// already raises a toast, and opening a sheet over whatever someone was doing
// -- mid-transfer, mid-property-browse -- is a worse interruption than a bar
// they can see and tap.

// A BID_PLACED that has been applied to the panel without a refetch.
//
// Bids are the one broadcast the room does not refetch on (PLAN.md section 3):
// a lot can take twenty of them and each one reaches every client, so the
// payload is applied directly instead. `kickedPlayerId` is captured from the
// auction the bid was accepted against -- NOT from the payload, which does not
// carry it -- because a deed can be the open lot of two different auctions, and
// a stale $120 from the first would otherwise show up over the second's $0
// opening.
export type LiveBid = {
  kickedPlayerId: string;
  propertyId: string;
  bidderId: string;
  amount: number;
};

const AuctionBar = ({
  room,
  allPlayers,
  availableProperties,
  currentPlayer,
  liveBid,
  onPlaceBid,
  onCloseAuction,
}: {
  room: Room;
  allPlayers: Player[];
  availableProperties: Property[];
  currentPlayer: Player;
  liveBid: LiveBid | null;
  onPlaceBid: (propertyId: string, amount: number) => void;
  onCloseAuction: (propertyId: string, kickedPlayerId: string) => void;
}) => {
  const stored = room?.auction;
  if (!stored) return null;

  // The overlay is applied by taking the larger bid rather than by clearing it
  // when the room refetches, and that is what makes the two sources safe to mix.
  // A bid only ever goes up within a lot, so `max` is monotone: a refetch that
  // lands between the broadcast and the write cannot walk the displayed high
  // bid backwards, and an overlay left over from a lot that has since closed is
  // ignored by the identity check instead of having to be cleaned up.
  const applies =
    liveBid !== null &&
    liveBid.kickedPlayerId === stored.kickedPlayerId &&
    liveBid.propertyId === stored.propertyId &&
    liveBid.amount > stored.highBid;
  const auction = applies
    ? {
        ...stored,
        highBid: liveBid.amount,
        highBidderId: liveBid.bidderId,
      }
    : stored;

  // Every deed in the room, by id. The lots stay assigned to the kicked player
  // for the length of the auction -- clearing them would put them in the Bank's
  // for-sale list at face price while they were still being bid on -- so both
  // the open lot and the queue resolve out of `players[].properties`, which the
  // room fetch already carries. `availableProperties` is folded in as well so
  // that a deed which has somehow reached the Bank still renders with a name
  // rather than disappearing.
  const deeds = new Map<string, Property>();
  for (const property of availableProperties ?? []) {
    deeds.set(property?.id, property);
  }
  for (const player of allPlayers ?? []) {
    for (const property of player?.properties ?? []) {
      deeds.set(property?.id, property);
    }
  }

  const lot = deeds.get(auction.propertyId);
  const upcoming = auction.queue
    .map((id) => deeds.get(id))
    .filter((property): property is Property => property !== undefined);
  const kickedPlayer = allPlayers?.find((p) => p?.id === auction.kickedPlayerId);
  const highBidder = auction.highBidderId
    ? allPlayers?.find((p) => p?.id === auction.highBidderId)
    : undefined;

  return (
    <Drawer>
      <DrawerTrigger asChild>
        <button
          type="button"
          className={`${josephinBold.className} flex w-full items-center justify-between gap-x-3 border-t border-white/20 bg-black px-4 py-2 text-left text-white transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-white`}
        >
          <span className={`flex min-w-0 items-baseline gap-x-2`}>
            <span
              className={`shrink-0 rounded-full border border-amber-300 px-2 py-[2px] text-[.6rem] uppercase tracking-wide text-amber-300`}
            >
              Auction
            </span>
            <span className={`truncate text-base`}>
              {lot?.name ?? "A deed"}
            </span>
          </span>
          <span className={`flex shrink-0 items-baseline gap-x-2 text-sm`}>
            <span className={`${josephinNormal.className} text-neutral-400`}>
              {auction.highBid > 0 ? `$${auction.highBid}` : "no bids"}
            </span>
            <span className={`text-amber-300`}>Bid</span>
          </span>
        </button>
      </DrawerTrigger>
      <DrawerContent className={`h-[90vh] border-x border-t bg-black`}>
        <DrawerTitle className={`${josephinBold.className} px-4 pb-2 text-white`}>
          {lot?.name ? `${lot.name} — up for auction` : "Up for auction"}
        </DrawerTitle>
        {/* Radix warns on a dialog with no description, and every drawer in
            this app is currently missing one. This one is not, because it was
            written after the warning was read rather than before; the other
            four are a separate job and not this change's to fold in. */}
        <DrawerDescription className={`sr-only`}>
          Bid on the open lot, or -- as the Banker -- close it.
        </DrawerDescription>
        {/* Keyed on the lot, so advancing to the next deed gives the panel a
            fresh instance instead of carrying a half-typed bid and an open
            confirm across the hammer. Both ids, because a deed can be the open
            lot of two different auctions. */}
        <AuctionPanel
          key={`${auction.kickedPlayerId}:${auction.propertyId}`}
          auction={auction}
          lot={lot}
          upcoming={upcoming}
          kickedPlayer={kickedPlayer}
          highBidder={highBidder}
          currentPlayer={currentPlayer}
          onPlaceBid={onPlaceBid}
          onCloseAuction={onCloseAuction}
        />
      </DrawerContent>
    </Drawer>
  );
};

export default AuctionBar;
