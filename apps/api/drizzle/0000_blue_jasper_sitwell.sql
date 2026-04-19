CREATE TABLE "exchange_rates" (
	"id" serial PRIMARY KEY NOT NULL,
	"base_currency" varchar(3) NOT NULL,
	"quote_currency" varchar(3) NOT NULL,
	"rate" numeric(16, 6) NOT NULL,
	"date" date NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "uq_exchange_rates_pair_date" UNIQUE("base_currency","quote_currency","date")
);
--> statement-breakpoint
CREATE INDEX "idx_exchange_rates_base_date" ON "exchange_rates" USING btree ("base_currency","date");