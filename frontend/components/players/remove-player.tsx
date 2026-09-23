import { useState } from "react";
import { Player } from "@/types/schema";
import { KickPlayerPayload } from "@/types/payloads";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Button } from "../ui/button";
import { CiCircleRemove } from "react-icons/ci";

type Disposition = KickPlayerPayload["disposition"];

// Every user-facing string in one place, because half of them change between
// kicking someone else and kicking yourself and reading the two versions side
// by side is the only way to keep the register consistent. Warm variant,
// chosen by Zach 2026-09-17 over a plain and a terse set (R7). The one line
// he did not choose from variants is `noSuccessors` -- the empty successor
// list was not in the ask.
const copyFor = (name: string, isSelf: boolean) => ({
  title: isSelf ? "Remove yourself?" : `Remove ${name}?`,
  body: isSelf
    ? "You'll be dropped from the game and won't be able to join back in. Someone else has to take the bank first."
    : `${name} will be dropped from the game and won't be able to join back in.`,
  dispositionHeading: isSelf
    ? "What happens to your properties?"
    : `What happens to ${name}'s properties?`,
  bank: "Return them to the Bank",
  auction: "Auction them off to the table",
  freeze: isSelf ? "Leave them with you" : `Leave them with ${name}`,
  successorHeading: "Hand the Bank to",
  noSuccessors: "There's no one else here to take the Bank.",
  confirm: isSelf ? "Remove myself" : `Remove ${name}`,
});

const OptionRow = ({
  label,
  selected,
  onSelect,
  swatch,
}: {
  label: string;
  selected: boolean;
  onSelect: () => void;
  swatch?: string;
}) => (
  <button
    type="button"
    onClick={onSelect}
    aria-pressed={selected}
    className={`flex w-full items-center gap-x-3 rounded-md border px-3 py-3 text-left text-base transition-colors
      focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black
      ${selected ? "border-black bg-black text-white" : "border-neutral-300 text-black hover:bg-black/5"}`}
  >
    {swatch && (
      <span
        aria-hidden
        style={{ backgroundColor: swatch }}
        className={`h-4 w-4 shrink-0 rounded-full border border-black`}
      />
    )}
    <span>{label}</span>
  </button>
);

// The banker-only control on a player card, plus its confirmation. Both live
// here rather than inline in `player-card-content.tsx` so that the card file
// carries one gated line instead of a second dialog; the gate itself stays at
// the call site, where it is greppable beside the other banker controls.
//
// Rendered on every card including the banker's own, because a banker removing
// themselves is the same action through the same successor-naming flow (D12) --
// it is deliberately NOT gated on the card-identity test that hides "Pay or
// Request" on your own card.
const RemovePlayer = ({
  player,
  currentPlayer,
  allPlayers,
  onKickPlayer,
}: {
  player: Player;
  currentPlayer: Player;
  allPlayers: Player[];
  onKickPlayer: (
    targetPlayerId: string,
    disposition: Disposition,
    successorPlayerId?: string,
  ) => void;
}) => {
  const [open, setOpen] = useState(false);
  // "Return to bank" preselected: the only disposition that leaves the board
  // fully playable (PLAN.md section 3).
  const [disposition, setDisposition] = useState<Disposition>("BANK");
  const [successorId, setSuccessorId] = useState<string | null>(null);

  const isSelf = currentPlayer?.id === player?.id;
  const copy = copyFor(player?.name, isSelf);

  // Only a banker target needs one, and the server refuses a successor named
  // for anyone else rather than dropping it silently (Phase 1, Zach 2026-09-17).
  const needsSuccessor = player?.isBanker === true;
  const successors = allPlayers.filter(
    (p) => p?.id !== player?.id && p?.isActive !== false,
  );

  const handleOpenChange = (next: boolean) => {
    if (next) {
      setDisposition("BANK");
      setSuccessorId(null);
    }
    setOpen(next);
  };

  const confirm = () => {
    if (needsSuccessor && !successorId) return;
    onKickPlayer(
      player?.id,
      disposition,
      needsSuccessor && successorId ? successorId : undefined,
    );
    setOpen(false);
  };

  return (
    <>
      <button
        type="button"
        onClick={() => handleOpenChange(true)}
        className={`flex items-center justify-between w-full mt-2 rounded-md px-1 transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black`}
      >
        <span>Remove player</span>
        <CiCircleRemove />
      </button>

      <Dialog open={open} onOpenChange={handleOpenChange}>
        {/* Pinned light deliberately -- see player-tags.tsx. The option rows
            below use border-neutral-300/text-black unselected and bg-black
            text-white selected, so a black card erases both the copy and the
            selection. Phase 3 restyles it and removes this bg-white. */}
        <DialogContent
          className={`sm:max-w-[425px] bg-white text-black max-h-[85vh] overflow-y-auto`}
        >
          <DialogHeader>
            <DialogTitle>{copy.title}</DialogTitle>
          </DialogHeader>

          <p className={`text-sm`}>{copy.body}</p>

          <div className={`flex flex-col gap-y-2`}>
            <p className={`text-sm`}>{copy.dispositionHeading}</p>
            <OptionRow
              label={copy.bank}
              selected={disposition === "BANK"}
              onSelect={() => setDisposition("BANK")}
            />
            <OptionRow
              label={copy.auction}
              selected={disposition === "AUCTION"}
              onSelect={() => setDisposition("AUCTION")}
            />
            <OptionRow
              label={copy.freeze}
              selected={disposition === "FREEZE"}
              onSelect={() => setDisposition("FREEZE")}
            />
          </div>

          {needsSuccessor && (
            <div className={`flex flex-col gap-y-2`}>
              <p className={`text-sm`}>{copy.successorHeading}</p>
              {successors.length === 0 ? (
                <p className={`text-sm`}>
                  {copy.noSuccessors}
                </p>
              ) : (
                successors.map((s) => (
                  <OptionRow
                    key={s?.id}
                    label={s?.name}
                    swatch={s?.color}
                    selected={successorId === s?.id}
                    onSelect={() => setSuccessorId(s?.id)}
                  />
                ))
              )}
            </div>
          )}

          <Button
            variant="destructive"
            onClick={confirm}
            disabled={needsSuccessor && !successorId}
          >
            {copy.confirm}
          </Button>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default RemovePlayer;
