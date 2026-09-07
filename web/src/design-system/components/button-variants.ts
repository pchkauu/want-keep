import { cva } from "class-variance-authority";

export const buttonVariants = cva("wk-button", {
  variants: {
    variant: {
      primary: "wk-button-primary",
      secondary: "wk-button-secondary",
      ghost: "wk-button-ghost",
      destructive: "wk-button-destructive",
      link: "wk-button-link",
    },
    size: {
      sm: "wk-button-sm",
      md: "wk-button-md",
      lg: "wk-button-lg",
      icon: "wk-button-icon",
    },
  },
  defaultVariants: { variant: "primary", size: "md" },
});
