## API

* 以下のAPIを追加
    * POST /ml/start
         * 機械学習の動作を開始する。
             * トレーニング
             * 推論
         * パラメータ
             * 入力データサイズ（データ範囲）
             * 学習回数
             * 学習済みネットワークのファイルのパス (undefinedでトレーニング実行)
             * 入力データ選択 (RAW or FFT)
    * GET /ml/stop
         * 機械学習の推論を停止する。
    
    * GET /acquisition/start
         * データ収集を開始する
    
    * GET /acquisition/stop
         * データ収集を停止する
    
    * GET /monitoring/start
         * モニタリング動作 (リアルタイム表示)を開始する
         * パラメータ
             * モニタリング有効チャネル

    * GET /monitoring/stop
         * モニタリング動作 (リアルタイム表示)を停止する
    
    * GET /acquisition/status
         * データ収集のステータスを取得する。

    * GET /acquisition/config/get
         * データ収集チャネルの構成を取得する

    * GET /acquisition/settings/get
         * データ収集チャネルの設定を取得する
    
    * POST /acquisition/settings/set
         * データ収集チャネルの設定を行う
         * ゲインとか
    
    * GET /status
         * 現在の状態を返す
             * データ収集の有効無効
             * ADCチャネルの状態
             * 機械学習の推論の有効無効
             * モニタリング動作の有効無

* WebSocket
    * ws:/monitoring/summary
        * モニタリングデータのサマリ (現在のadc-dbが出力しているデータ)を取得できる
        * バイナリ 
    * ws:/ml/event
        * 機械学習の推論に関するイベント通知
            * training-progress
                * トレーニング中
            * training-done
                * トレーニング完了
            * inference-result
                *　推論結果

## 機械学習サーバの接続設定

* Oliveのローカルに設定ファイルを置いて対応。
* こんなかんじ？

```json
{
    "default": "Local",
    "connections":
    [
        {
            "name": "Local",
            "connection": {
                "type": "tcp",
                "address": "127.0.0.1",
                "port": "1000"
            }
        },
        {
            "name": "External",
            "connection": {
                "type": "tcp",
                "address": "127.0.0.1",
                "port": "2000"
            }
        }
    ]
}
```

## OS-ELMサーバの仕様
* 起動すると、指定したポートで待ち受け。
* 指定したユニット数*1.1だけ、トレーニング
    * その間、loss値として-1を返す。
* トレーニングが完了すると実際のloss値を返す。

## api-server の内部構造の変更

* 現状、ML Serverへリアルタイムデータを流す経路がない。
* adc-dmaから受け取ったリアルタイムデータをWebSocket経由で送信している (`realtime`パッケージ/realtime.Server, realtime.Client)
* -> ML Serverも realtime.Clientと同じように扱われるようにして、realtime.Serverに登録する
* realtime.Server -[realtime data (data + FFT)]-> mlserver.adapter -[input data]-> mlserver.MLServer -[loss value]-> mlserver.WSServer

* realtimeサーバは `adc-db` (実行可能ファイル名はadc-dma) からパイプ経由でデータを受け取っている。
    * <s>データ構造の詳細は Header-Specification.txt に記載されているようなので、それを参考に受信データのパーサを実装</s>
    * ↑はデータファイルのヘッダの情報なので間違い。正しくはソフト仕様書の3.2.6に書いてあるとおり。
    * 生データがないので、生データとトリガ情報を送るようにadc-dbを変更する。


 