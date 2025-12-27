# go-microps

「ゼロからのTCP/IPプロトコルスタック自作入門」をGo言語で実装する。自習用リポジトリ。  

## 概要

「[ゼロからのTCP/IPプロトコルスタック自作入門](https://book.mynavi.jp/ec/products/detail/id=149014)」を読み進めながら、
TCP/IP プロトコルスタック microps を Go 言語で実装する。  

## はじめに
### 目的

- ネットワークの理解を深める。
- Go 言語の学習。

## 実装環境

- Ubuntu 24.03.0 LTS (WSL2)
- go 1.22.2

## 方針

- 書籍では、予めいくつかの定数やユーティリティ関数が用意されているが、それらは必要になったら実装していく。
- なるべく、Go言語らしいコードを心掛ける。 Unsafe は極力使わない。
- コードや説明文には日本語を使用する。

## 開発記録

### Step00: はじめに

- Makefile
    - 最低限のものを用意。
    - ./test ディレクトリ内のコードをビルド対象とする。
    - ビルドする際にlintを掛けるようにしておく。
- platform_linux.go
    - プラットフォーム依存のコードは言語側で吸収してくれそうだが、一応作成しておく。後で不要になるかもしれない。
    - Goのビルドタグ機能を使うため末尾に _linux を付ける。
    - メモリ確保コードは Go では不要、ロック機構は sync.Mutex を使用するので不要、疑似乱数も rand パッケージを使用するので不要。
    - 結果として、platform_xxx 系の関数だけを用意するが、乱数シードの初期化も Go では不要なので、関数のスケルトンだけ用意することになった。
- util.go
    - internal/util パッケージとする。
    - ロギング関数群
        - FILE* は Go だと io.Writer。
        - flockfile(), funlockfile() は sync.Mutex で代用。
        - \_\_FILE\_\_ や \_\_LINE\_\_ といったマクロ定数は、 runtime.Caller で代用。
    - HexDump の中では Unsafe を使う必要がある。Unsafe を使うのはここだけの予定。
    - HexDump の引数は any 型だが、ダンプできるのは固定長型のみ。
        - string 型は渡せない。ascii 配列に変換する。
        - スライス型は渡せない。固定長の配列型にする。
        - int 型は int8 などサイズが明確な型にする。
- dump_on.go, dump_off.go
    - 書籍では CFLAGS="-DHEXDUMP" を付けたときだけダンプ出力するような仕掛けになっているので、これを Go のビルドスイッチで再現する。
- net.go
    - 書籍通りに実装。
- test.go
    - テストコードは main パッケージとする。
    - Go では配列を定数にできないので testData は変数にする。
    - シグナルの割り込み処理は signal.NotifyContext を利用する。
        - defer stop() を呼び出すため、処理を setup() から main() に移した。
        - setup() の中で割り込み処理を書くが、onSignal() のような関数は設けずに無名関数を Go ルーチンで呼び出す。
