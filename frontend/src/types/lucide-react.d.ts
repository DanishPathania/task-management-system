import * as React from "react";

declare module "lucide-react" {
  export interface LucideProps extends React.SVGProps<SVGSVGElement> {
    size?: string | number;
    color?: string;
    strokeWidth?: string | number;
    absoluteStrokeWidth?: boolean;
    className?: string;
  }

  export type LucideIcon = React.ForwardRefExoticComponent<
    LucideProps & React.RefAttributes<SVGSVGElement>
  >;

  export const createLucideIcon: (iconName: string, iconNode: [string, Record<string, string>][]) => LucideIcon;

  export const AlertCircle: LucideIcon;
  export const ArrowRight: LucideIcon;
  export const ArrowUpRight: LucideIcon;
  export const Calendar: LucideIcon;
  export const CheckCircle2: LucideIcon;
  export const CheckSquare: LucideIcon;
  export const Clock: LucideIcon;
  export const Filter: LucideIcon;
  export const FolderKanban: LucideIcon;
  export const Lock: LucideIcon;
  export const LogOut: LucideIcon;
  export const Mail: LucideIcon;
  export const Plus: LucideIcon;
  export const Send: LucideIcon;
  export const Shield: LucideIcon;
  export const Sparkles: LucideIcon;
  export const Trash2: LucideIcon;
  export const User: LucideIcon;
  export const UserIcon: LucideIcon;
  export const UserPlus: LucideIcon;
  export const Users: LucideIcon;
  export const X: LucideIcon;
  export const Zap: LucideIcon;

  export default LucideIcon;
}
