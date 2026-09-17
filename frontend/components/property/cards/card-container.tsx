import { josephinBold } from "@/components/ui/fonts";

const PropertyCardContainer = ({
  onClick,
  className,
  children,
}: {
  onClick?: () => void;
  className?: string;
  children: React.ReactNode;
}) => {
  const classes = `bg-white border p-4 text-black border-black property-card-aspect-ratio w-64 ${josephinBold.className} ${className}`;

  // A deed with no handler is not a control. Rendering one as a `button`
  // anyway put a focusable, do-nothing card in the tab order of the read-only
  // property drawer. `text-center` keeps the inherited alignment a `button`
  // would have given it. Every other caller passes `onClick`, so this is the
  // read-only path only.
  if (!onClick) {
    return (
      <div className={`text-center ${classes}`}>
        <div className={`border border-black h-full p-2 relative`}>
          {children}
        </div>
      </div>
    );
  }

  return (
    <>
      <button className={classes} onClick={onClick}>
        <div className={`border border-black h-full p-2 relative`}>
          {children}
        </div>
      </button>
    </>
  );
};

export default PropertyCardContainer;
