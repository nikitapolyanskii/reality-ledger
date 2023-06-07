library(ggplot2)
library(dplyr)
library(tidyr)
library("reshape2")


###########################
#
#
#  Visualisation of getRealityTime.txt
#
#
###########################

df <- data.frame(numberOfConflicts = numeric(),
                 time = numeric(),
                 size = numeric())

numberOfSims <- read.table("getRealityTime.txt", nrows = 1)$V1

for (i in 1:numberOfSims) {
  read.table("getRealityTime.txt",
             skip = 1 + ((i - 1) * 1001),
             nrows = 1) -> simData
  read.table("getRealityTime.txt",
             skip = 2 + ((i - 1) * 1001),
             nrows = 1000) -> dfRead
  colnames(dfRead) <- c("time", "size")
  dfRead$numberOfConflicts <- simData$V1
  df <- rbind(df, dfRead)
}

colnames(df) <- c("time", "size", "conflicts")

df$conflicts <- as.factor(df$conflicts)
p1 <- ggplot(df, aes(x = time, fill = conflicts)) +
  geom_density(alpha = 0.7) +
  labs(x = "time (s)", y = "density", title = "Time to compute the preferred reality")

p1
ggsave(
  p1,
  filename = "TimeToCompute.png",
  width = 24,
  height = 12,
  units = "cm",
  dpi = 300
)



p2 <- ggplot(df, aes(x = size, fill = conflicts)) +
  geom_density(alpha = 0.7) +
  labs(x = "size", y = "density", title = "Size of the preferred reality")

p2
ggsave(
  p2,
  filename = "SizeOfReality.png",
  width = 24,
  height = 12,
  units = "cm",
  dpi = 300
)


###########################
#
#
#  Visualisation of ledgerGrow.txt
#
#
###########################


dfLedgerGrowth <- data.frame(
  numberOfConflicts = numeric(),
  computeBranch = character(),
  probabilityOfConflict = numeric(),
  timeStamp = numeric(),
  numberTx = numeric(),
  numberConflicts = numeric()
)

numberOfSims <- read.table("ledgerGrow.txt", nrows = 1)$V1 * 2
lineCounter = 1
for (i in 1:numberOfSims) {
  read.table("ledgerGrow.txt", skip =  lineCounter, nrows = 1) -> simData
  lineCounter <- lineCounter + 1
  nRows2Read <- simData$V3
  read.table("ledgerGrow.txt", skip = lineCounter, nrows = nRows2Read) -> dfRead
  lineCounter <- lineCounter + nRows2Read
  colnames(dfRead) <- c("timeStamp", "numberTx", "numberConflicts")
  dfRead$computeBranch <- simData$V2
  dfRead$probabilityOfConflict <- simData$V1
  dfLedgerGrowth <- rbind(dfLedgerGrowth, dfRead)
}

dfLedgerGrowth <- dfLedgerGrowth %>%
  mutate(
    computeBranch = ifelse(
      computeBranch == "true",
      "with computing branches",
      "without computing branches"
    )
  )


dfPlot <- dfLedgerGrowth
colnames(dfPlot) <-
  c("time", "transactions", "conflicts", "branches", "probability")
dfPlot$probability <- as.factor(dfPlot$probability)
p3 <- ggplot(dfPlot, aes(y = time, x = transactions)) +
  geom_line(aes(color = probability)) +
  facet_wrap( ~ branches, ncol = 1) +
  labs(x = "number of transactions", y = "time (s)", title = "Time to handle transactions")
p3

ggsave(
  p3,
  filename = "TimeToHandleTransactions.png",
  width = 24,
  height = 12,
  units = "cm",
  dpi = 300
)



###########################
#
#
#  Visualisation of ledgerGrowAndPrune.txt
#
#
###########################

dfLedgerGrowthWithPrune <- read.table("ledgerGrowAndPrune.txt")

colnames(dfLedgerGrowthWithPrune) <-
  c("time", "TxsRAM", "ConflictsRAM",
    "ConfirmedTxs")

dfPlot <- dfLedgerGrowthWithPrune %>%
  gather(type, number, TxsRAM, ConflictsRAM, ConfirmedTxs)


p4 <- ggplot(dfPlot, aes(x = time, y = number)) +
  geom_line()


type_names <- c(
  `TxsRAM` = "Transactions in RAM",
  `ConflictsRAM` = "Conflicts in RAM",
  `ConfirmedTxs` = "Confirmed Transactions"
)

p4 <-
  p4 + facet_wrap(
    ~ type,
    ncol = 1,
    scales = "free",
    labeller = as_labeller(type_names)
  ) +
  scale_y_continuous(
    labels = function(x)
      format(x, scientific = TRUE)
  ) +
  labs(x = "time (s)", y = "number", title = "Growth of ledger with pruning") 
  
  p4


ggsave(
  p4,
  filename = "LedgerGrowth.png",
  width = 24,
  height = 16,
  units = "cm",
  dpi = 300
)

