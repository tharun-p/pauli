"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Menu,
  Scale,
  Shield,
  Timer,
  Wallet,
  X,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { type ReactNode, useEffect, useState } from "react";

const nav = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/validators", label: "Validators", icon: Shield },
  { href: "/epochs", label: "Epochs", icon: Timer },
  { href: "/rewards", label: "Rewards", icon: Wallet },
  { href: "/balances", label: "Balances", icon: Scale },
] as const;

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  useEffect(() => {
    setMobileNavOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!mobileNavOpen) {
      document.body.style.overflow = "";
      return;
    }
    const mq = window.matchMedia("(max-width: 767px)");
    const syncScrollLock = () => {
      document.body.style.overflow = mq.matches ? "hidden" : "";
    };
    syncScrollLock();
    mq.addEventListener("change", syncScrollLock);
    return () => {
      mq.removeEventListener("change", syncScrollLock);
      document.body.style.overflow = "";
    };
  }, [mobileNavOpen]);

  return (
    <div className="flex min-h-screen">
      <button
        type="button"
        tabIndex={mobileNavOpen ? 0 : -1}
        aria-hidden={!mobileNavOpen}
        aria-label="Close menu"
        className={cn(
          "fixed inset-0 z-[50] bg-black/50 backdrop-blur-sm transition-opacity md:hidden",
          mobileNavOpen ? "opacity-100" : "pointer-events-none opacity-0",
        )}
        onClick={() => setMobileNavOpen(false)}
      />

      <header className="fixed top-0 left-0 right-0 z-[55] flex h-[calc(3.5rem+env(safe-area-inset-top,0px))] items-center border-b border-border bg-background/95 px-4 pt-[env(safe-area-inset-top,0px)] backdrop-blur-xl md:hidden">
        <span className="font-heading text-lg font-bold tracking-tight text-primary">Pauli</span>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-expanded={mobileNavOpen}
          aria-controls="app-shell-nav"
          aria-label={mobileNavOpen ? "Close navigation" : "Open navigation"}
          onClick={() => setMobileNavOpen((open) => !open)}
        >
          {mobileNavOpen ? <X className="size-5" /> : <Menu className="size-5" />}
        </Button>
      </header>

      <aside
        id="app-shell-nav"
        className={cn(
          "fixed left-0 top-0 z-[60] flex h-screen w-64 flex-col border-r border-sidebar-border bg-sidebar backdrop-blur-xl transition-transform duration-200 ease-out",
          mobileNavOpen ? "translate-x-0" : "-translate-x-full md:translate-x-0",
        )}
      >
        <div className="p-8 pt-[max(2rem,calc(env(safe-area-inset-top,0px)+1rem))] md:pt-8">
          <h1 className="font-heading text-2xl font-bold tracking-tighter text-primary">Pauli</h1>
          <p className="font-label mt-1 text-[10px] font-medium uppercase tracking-[0.2em] text-muted-foreground">
            Ethereum validator OS
          </p>
        </div>
        <nav className="flex flex-1 flex-col gap-2 px-3">
          {nav.map(({ href, label, icon: Icon }) => {
            const active =
              href === "/dashboard"
                ? pathname === "/dashboard"
                : pathname === href || pathname.startsWith(`${href}/`);
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  "flex items-center gap-3 rounded-xl border px-4 py-3 text-sm font-medium transition-all duration-200",
                  active
                    ? "border-primary/50 bg-primary/10 text-primary shadow-[0_0_0_1px_rgba(223,255,0,0.18)]"
                    : "border-transparent text-muted-foreground hover:border-border/60 hover:bg-muted/40 hover:text-foreground",
                )}
              >
                <Icon className="size-5 shrink-0 opacity-90" strokeWidth={1.75} />
                {label}
              </Link>
            );
          })}
        </nav>
        <div className="p-4 pb-[max(1rem,calc(env(safe-area-inset-bottom,0px)+0.75rem))]">
          <div className="rounded-lg border border-border/60 bg-muted/30 p-3 text-xs text-muted-foreground">
            Pauli read API. Configure{" "}
            <code className="text-primary/90">PAULI_API_ORIGIN</code> for rewrites.
          </div>
        </div>
      </aside>

      <main className="min-w-0 flex-1 p-4 pt-[calc(3.5rem+env(safe-area-inset-top,0px)+1rem)] md:ml-64 md:p-8 md:pt-8">
        {children}
      </main>
    </div>
  );
}
