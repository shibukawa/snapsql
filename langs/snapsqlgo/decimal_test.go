// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapsqlgo

import (
	"fmt"
	"log"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/shopspring/decimal"
)

func TestDecimal(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	env, err := cel.NewEnv(
		DecimalLibrary,
		cel.Variable("a", DecimalType),
		cel.Variable("b", DecimalType),
	)
	if err != nil {
		log.Fatalf("CEL環境の作成に失敗しました: %v", err)
	}

	// CEL式をパース
	// price と limit がカスタムのDecimal型として扱われるため、直接比較可能
	expression := `a` // double() などの変換なしで直接比較
	ast, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		log.Fatalf("CEL式のパースに失敗しました: %v", issues.Err())
	}

	// 型チェック
	checkedAST, issues := env.Check(ast)
	if issues != nil && issues.Err() != nil {
		log.Fatalf("CEL式のチェックに失敗しました: %v", issues.Err())
	}
	if checkedAST.OutputType() != DecimalType {
		log.Fatalf("CEL式の戻り値の型が期待されるDecimalではありません: %v", checkedAST.OutputType())
	}

	// プログラムを作成
	program, err := env.Program(checkedAST)
	if err != nil {
		log.Fatalf("CELプログラムの作成に失敗しました: %v", err)
	}

	// decimal.Decimal 型の値を準備
	productPrice := decimal.NewFromFloat(19.99)
	priceLimit := decimal.NewFromFloat(20.00)

	// CEL式を評価
	// カスタムラッパー型 `Decimal` のインスタンスを渡す
	vars := map[string]interface{}{
		"a": &Decimal{productPrice}, // ラッパーで包んでポインタを渡す
		"b": &Decimal{priceLimit},
	}

	result, _, err := program.Eval(vars)
	if err != nil {
		log.Fatalf("CEL式の評価に失敗しました: %v", err)
	}
	fmt.Printf("取得結果: %v\n", result)                                          // false
	fmt.Printf("評価結果: %v\n", result.(*Decimal).Equal(&Decimal{productPrice})) // true
}
