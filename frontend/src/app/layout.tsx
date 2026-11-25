import "@/styles/globals.css";
import { AuthProvider } from "@/hooks/useAuth";
import { Navbar } from "@/components/layout/Navbar";

export const metadata = {
  title: "CollabTask - Real-Time Collaborative Task Management",
  description: "Production-ready real-time collaborative task management system with WebSockets, Go, and Next.js",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="bg-slate-950 text-slate-100 min-h-screen antialiased flex flex-col">
        <AuthProvider>
          <Navbar />
          <main className="flex-1 flex flex-col">
            {children}
          </main>
        </AuthProvider>
      </body>
    </html>
  );
}
