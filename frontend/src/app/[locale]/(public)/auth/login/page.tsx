"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link, useRouter } from "@/i18n/routing";
import { useAuth } from "@/lib/auth";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Alert } from "@/components/ui/alert";
import { GoogleSignIn } from "@/components/auth/google-sign-in";
import { UniversityLogo } from "@/components/brand/university-logo";

export default function LoginPage() {
  const t = useTranslations("auth");
  const ts = useTranslations("site");
  const router = useRouter();
  const { setAuth } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await api.post<{
        token: string;
        user: {
          id: string;
          email: string;
          name_ar?: string;
          name_en?: string;
          role: string;
          institution_id?: string;
          college_id?: string;
          department_id?: string;
        };
      }>("/api/v1/auth/login", { email, password });
      if (res.data) {
        setAuth(res.data.token, res.data.user);
        router.push("/portal");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto w-full max-w-md">
      <Card className="w-full border-t-4 border-t-primary shadow-lg">
        <CardHeader className="items-center text-center">
          <Link href="/" className="mx-auto rounded-md focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
            <UniversityLogo variant="color" height={96} alt={ts("logoAlt")} priority />
          </Link>
          <CardTitle className="mt-3 text-xl text-primary">{t("portalName")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("universityName")}</p>
        </CardHeader>
        <CardContent className="space-y-6">
          {error && <Alert variant="danger">{error}</Alert>}
          <GoogleSignIn onError={setError} />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            className="block w-full text-center text-xs text-muted-foreground hover:underline"
          >
            {showPassword ? t("hidePasswordLogin") : t("passwordLogin")}
          </button>
          {showPassword && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">{t("email")}</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  dir="ltr"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">{t("password")}</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  dir="ltr"
                />
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "..." : t("login")}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
